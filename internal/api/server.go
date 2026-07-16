package api

import (
	"bytes"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/azhry/nala-chores/internal/k8s"
	"github.com/azhry/nala-chores/internal/runner"
	"github.com/azhry/nala-chores/internal/store"
)

//go:embed web/*
var webAssets embed.FS

type Config struct {
	Namespace     string
	Image         string
	SecretName    string
	ApplyJobs     bool
	KubectlPath   string
	DefaultAPIURL string
	StatePath     string
}

type Server struct {
	store *store.Memory
	cfg   Config
	log   *slog.Logger
}

func NewServer(cfg Config, log *slog.Logger) *Server {
	if log == nil {
		log = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	if cfg.Namespace == "" {
		cfg.Namespace = "agent-runner"
	}
	if cfg.Image == "" {
		cfg.Image = "opencode-runner-backend:local"
	}
	if cfg.SecretName == "" {
		cfg.SecretName = "runner-secrets"
	}
	if cfg.KubectlPath == "" {
		cfg.KubectlPath = "kubectl"
	}
	return &Server{store: store.NewMemory(cfg.StatePath), cfg: cfg, log: log}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /", s.webHandler())
	mux.HandleFunc("POST /configs", s.saveConfig)
	mux.HandleFunc("GET /configs", s.listConfigs)
	mux.HandleFunc("GET /configs/{config_id}", s.getConfig)
	mux.HandleFunc("POST /runs", s.createRun)
	mux.HandleFunc("GET /runs", s.listRuns)
	mux.HandleFunc("GET /runs/{request_id}", s.getRun)
	mux.HandleFunc("GET /runs/{request_id}/logs", s.getRunLogs)
	mux.HandleFunc("POST /runs/{request_id}/stop", s.stopRun)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	return requestLogger(s.log, mux)
}

func (s *Server) saveConfig(w http.ResponseWriter, r *http.Request) {
	var input runner.ConfigurationInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	if input.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if input.RepoURL == "" {
		writeError(w, http.StatusBadRequest, "repo_url is required")
		return
	}
	cfg, err := s.store.SaveConfig(input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if s.cfg.ApplyJobs {
		if err := s.syncConfigSecret(cfg.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (s *Server) listConfigs(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, runner.ConfigurationList{Configurations: s.store.ListConfigs()})
}

func (s *Server) getConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.store.GetConfig(r.PathValue("config_id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "configuration not found")
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (s *Server) createRun(w http.ResponseWriter, r *http.Request) {
	var req runner.RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	if err := validateRequest(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.RequestID == "" {
		req.RequestID = makeRequestID(req.ConfigID)
	}

	run, created, err := s.store.Create(req)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "configuration not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !created {
		writeJSON(w, http.StatusOK, run)
		return
	}

	go s.provision(run.RequestID)
	writeJSON(w, http.StatusAccepted, run)
}

func (s *Server) getRun(w http.ResponseWriter, r *http.Request) {
	run, err := s.store.Get(r.PathValue("request_id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (s *Server) getRunLogs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("request_id")
	run, err := s.store.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}
	if run.Logs != "" {
		writeJSON(w, http.StatusOK, map[string]string{"logs": run.Logs})
		return
	}
	if run.JobName == "" {
		writeJSON(w, http.StatusOK, map[string]string{"logs": "Job has not been created yet."})
		return
	}

	tail := r.URL.Query().Get("tail")
	if tail == "" {
		tail = "200"
	}
	if _, err := strconv.Atoi(tail); err != nil {
		writeError(w, http.StatusBadRequest, "tail must be an integer")
		return
	}

	cmd := exec.Command(s.cfg.KubectlPath, "-n", s.cfg.Namespace, "logs", "job/"+run.JobName, "--tail", tail)
	out, err := cmd.CombinedOutput()
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{
			"logs": fmt.Sprintf("Logs are not available yet.\n%s", strings.TrimSpace(string(out))),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"logs": string(out)})
}

func (s *Server) listRuns(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			writeError(w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}
		limit = parsed
	}
	if configID := r.URL.Query().Get("config_id"); configID != "" {
		writeJSON(w, http.StatusOK, runner.RunList{Runs: s.store.ListByConfig(configID, limit)})
		return
	}
	writeJSON(w, http.StatusOK, runner.RunList{Runs: s.store.List(limit)})
}

func (s *Server) stopRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("request_id")
	run, err := s.store.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}

	if s.cfg.ApplyJobs && run.JobName != "" {
		cmd := exec.Command(s.cfg.KubectlPath, "-n", s.cfg.Namespace, "delete", "job", run.JobName, "--ignore-not-found=true")
		if out, err := cmd.CombinedOutput(); err != nil {
			writeError(w, http.StatusBadGateway, fmt.Sprintf("kubectl delete failed: %s: %s", err, strings.TrimSpace(string(out))))
			return
		}
	}

	now := time.Now().UTC()
	run, _ = s.store.Update(id, func(run *runner.Run) {
		run.Phase = runner.PhaseCancelled
		run.Message = "cancelled by API request"
		run.CompletedAt = &now
	})
	s.captureLogs(id, run.JobName)
	writeJSON(w, http.StatusOK, run)
}

func (s *Server) provision(id string) {
	run, err := s.store.Update(id, func(run *runner.Run) {
		run.Phase = runner.PhaseProvisioning
		run.JobName = k8s.JobName(run.RequestID)
		run.LogsCommand = fmt.Sprintf("kubectl -n %s logs -f job/%s", s.cfg.Namespace, run.JobName)
	})
	if err != nil {
		s.log.Error("update run before provisioning", "error", err)
		return
	}

	secretName := s.cfg.SecretName
	if run.ConfigID != "" {
		secretName = configSecretName(run.ConfigID)
	}

	manifest, err := k8s.RenderJob(run, k8s.JobOptions{
		Namespace:  s.cfg.Namespace,
		Image:      s.cfg.Image,
		SecretName: secretName,
	})
	if err != nil {
		s.fail(id, err)
		return
	}

	if !s.cfg.ApplyJobs {
		_, _ = s.store.Update(id, func(run *runner.Run) {
			run.Phase = runner.PhaseQueued
			run.Message = "dry-run mode: job manifest rendered but not applied"
		})
		s.log.Info("rendered dry-run job", "request_id", id, "manifest", string(manifest))
		return
	}

	cmd := exec.Command(s.cfg.KubectlPath, "apply", "-f", "-")
	cmd.Stdin = bytes.NewReader(manifest)
	out, err := cmd.CombinedOutput()
	if err != nil {
		s.fail(id, fmt.Errorf("kubectl apply failed: %w: %s", err, strings.TrimSpace(string(out))))
		return
	}

	_, _ = s.store.Update(id, func(run *runner.Run) {
		run.Phase = runner.PhaseSetup
		run.Message = strings.TrimSpace(string(out))
	})
	go s.watchJob(id, run.JobName)
}

func (s *Server) fail(id string, err error) {
	now := time.Now().UTC()
	_, updateErr := s.store.Update(id, func(run *runner.Run) {
		run.Phase = runner.PhaseFailed
		run.Message = err.Error()
		run.CompletedAt = &now
	})
	if updateErr != nil && !errors.Is(updateErr, store.ErrNotFound) {
		s.log.Error("mark run failed", "request_id", id, "error", updateErr)
	}
}

func (s *Server) watchJob(id, jobName string) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	deadline := time.After(12 * time.Hour)
	for {
		select {
		case <-ticker.C:
			phase, msg, done := s.jobPhase(jobName)
			if phase != "" {
				_, _ = s.store.Update(id, func(run *runner.Run) {
					run.Phase = phase
					run.Message = msg
					if done {
						now := time.Now().UTC()
						run.CompletedAt = &now
					}
				})
			}
			if done {
				s.captureLogs(id, jobName)
				return
			}
		case <-deadline:
			s.fail(id, errors.New("timed out waiting for Kubernetes Job"))
			return
		}
	}
}

func (s *Server) jobPhase(jobName string) (runner.Phase, string, bool) {
	cmd := exec.Command(s.cfg.KubectlPath, "-n", s.cfg.Namespace, "get", "job", jobName, "-o", "json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Sprintf("kubectl get job failed: %s", strings.TrimSpace(string(out))), false
	}

	var job struct {
		Status struct {
			Active     int `json:"active"`
			Succeeded  int `json:"succeeded"`
			Failed     int `json:"failed"`
			Conditions []struct {
				Type    string `json:"type"`
				Status  string `json:"status"`
				Reason  string `json:"reason"`
				Message string `json:"message"`
			} `json:"conditions"`
		} `json:"status"`
	}
	if err := json.Unmarshal(out, &job); err != nil {
		return "", fmt.Sprintf("parse Kubernetes Job status: %s", err), false
	}
	for _, condition := range job.Status.Conditions {
		if condition.Status != "True" {
			continue
		}
		switch condition.Type {
		case "Complete":
			return runner.PhaseSucceeded, firstNonEmpty(condition.Message, "job completed"), true
		case "Failed":
			return runner.PhaseFailed, firstNonEmpty(condition.Message, condition.Reason, "job failed"), true
		}
	}
	if job.Status.Active > 0 {
		return runner.PhaseEditing, "sandbox pod is running", false
	}
	return runner.PhaseProvisioning, "waiting for sandbox pod", false
}

func (s *Server) syncConfigSecret(configID string) error {
	secret, err := s.store.GetConfigSecret(configID)
	if err != nil {
		return err
	}
	secretName := configSecretName(configID)
	data := map[string]string{}
	if secret.GitHubToken != "" {
		data["git_token"] = secret.GitHubToken
	}
	if secret.OpenCodeAPIKey != "" {
		data["opencode_api_key"] = secret.OpenCodeAPIKey
	}
	if secret.LinearAPIKey != "" {
		data["linear_api_key"] = secret.LinearAPIKey
	}
	if len(data) == 0 {
		return nil
	}

	manifest := map[string]any{
		"apiVersion": "v1",
		"kind":       "Secret",
		"metadata": map[string]string{
			"name":      secretName,
			"namespace": s.cfg.Namespace,
		},
		"type": "Opaque",
		"data": encodeSecretData(data),
	}
	body, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	cmd := exec.Command(s.cfg.KubectlPath, "apply", "-f", "-")
	cmd.Stdin = bytes.NewReader(body)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kubectl apply config secret failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func configSecretName(configID string) string {
	clean := strings.ToLower(configID)
	var b strings.Builder
	for _, r := range clean {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	name := strings.Trim(b.String(), "-")
	if name == "" {
		name = "default"
	}
	if len(name) > 45 {
		name = strings.TrimRight(name[:45], "-")
	}
	return "runner-config-" + name
}

func encodeSecretData(values map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range values {
		out[key] = base64.StdEncoding.EncodeToString([]byte(value))
	}
	return out
}

func (s *Server) captureLogs(id, jobName string) {
	if jobName == "" || !s.cfg.ApplyJobs {
		return
	}
	cmd := exec.Command(s.cfg.KubectlPath, "-n", s.cfg.Namespace, "logs", "job/"+jobName, "--tail", "1000")
	out, err := cmd.CombinedOutput()
	logs := string(out)
	if err != nil && strings.TrimSpace(logs) == "" {
		logs = "Logs were not available when the run completed."
	}
	_, _ = s.store.Update(id, func(run *runner.Run) {
		run.Logs = logs
	})
}

func validateRequest(req runner.RunRequest) error {
	switch {
	case req.ConfigID == "" && req.RequestID == "":
		return errors.New("request_id is required when config_id is not supplied")
	case req.ConfigID == "" && req.RepoURL == "":
		return errors.New("repo_url is required")
	case req.ConfigID == "" && req.SourceBranch == "":
		return errors.New("source_branch is required")
	case req.Prompt == "":
		return errors.New("prompt is required")
	default:
		return nil
	}
}

func makeRequestID(configID string) string {
	prefix := "run"
	if configID != "" {
		prefix = configID
	}
	clean := strings.ToLower(prefix)
	var b strings.Builder
	for _, r := range clean {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	prefix = strings.Trim(b.String(), "-")
	if prefix == "" {
		prefix = "run"
	}
	return fmt.Sprintf("%s-%d", prefix, time.Now().UTC().Unix())
}

func requestLogger(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Info("request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *Server) webHandler() http.Handler {
	sub, err := fs.Sub(webAssets, "web")
	if err != nil {
		panic(err)
	}
	return http.FileServer(http.FS(sub))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
