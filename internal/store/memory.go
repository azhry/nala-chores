package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/azhry/nala-chores/internal/runner"
)

var ErrNotFound = errors.New("not found")

type Memory struct {
	mu     sync.RWMutex
	path   string
	runs   map[string]runner.Run
	config map[string]storedConfig
}

type storedConfig struct {
	Public         runner.Configuration `json:"public"`
	GitHubToken    string               `json:"github_token,omitempty"`
	OpenCodeAPIKey string               `json:"opencode_api_key,omitempty"`
	KiloAPIKey     string               `json:"kilo_api_key,omitempty"`
	LinearAPIKey   string               `json:"linear_api_key,omitempty"`
}

type diskState struct {
	Runs           map[string]runner.Run   `json:"runs"`
	Configurations map[string]storedConfig `json:"configurations"`
}

func NewMemory(path string) *Memory {
	s := &Memory{
		path:   path,
		runs:   map[string]runner.Run{},
		config: map[string]storedConfig{},
	}
	if path != "" {
		_ = s.load()
	}
	return s
}

func (s *Memory) SaveConfig(input runner.ConfigurationInput) (runner.Configuration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	id := input.ID
	if id == "" {
		id = slug(input.Name)
	}
	if id == "" {
		id = fmt.Sprintf("config-%d", now.Unix())
	}

	existing := s.config[id]
	cfg := existing.Public
	if cfg.CreatedAt.IsZero() {
		cfg.CreatedAt = now
	}
	cfg.ID = id
	cfg.Name = defaultString(input.Name, id)
	cfg.RepoURL = input.RepoURL
	cfg.SourceBranch = defaultString(input.SourceBranch, "main")
	cfg.WorkDirectory = defaultString(input.WorkDirectory, ".")
	cfg.HarnessRepoURL = input.HarnessRepoURL
	cfg.AgentProvider = defaultString(input.AgentProvider, "opencode")
	cfg.AgentModel = defaultAgentModel(cfg.AgentProvider, input.AgentModel)
	cfg.HarnessName = defaultString(input.HarnessName, "default")
	cfg.SandboxSize = defaultString(input.SandboxSize, "large")
	cfg.ConfigPath = input.ConfigPath
	cfg.CreateMR = input.CreateMR
	cfg.PushChanges = input.PushChanges || input.CreateMR
	cfg.UpdatedAt = now

	secret := existing
	secret.Public = cfg
	if input.GitHubToken != "" {
		secret.GitHubToken = input.GitHubToken
	}
	if input.OpenCodeAPIKey != "" {
		secret.OpenCodeAPIKey = input.OpenCodeAPIKey
	}
	if input.KiloAPIKey != "" {
		secret.KiloAPIKey = input.KiloAPIKey
	}
	if input.LinearAPIKey != "" {
		secret.LinearAPIKey = input.LinearAPIKey
	}
	if input.ClearGitHubToken {
		secret.GitHubToken = ""
	}
	if input.ClearOpenCodeAPIKey {
		secret.OpenCodeAPIKey = ""
	}
	if input.ClearKiloAPIKey {
		secret.KiloAPIKey = ""
	}
	if input.ClearLinearAPIKey {
		secret.LinearAPIKey = ""
	}
	secret.Public.HasGitHubToken = secret.GitHubToken != ""
	secret.Public.HasOpenCodeAPIKey = secret.OpenCodeAPIKey != ""
	secret.Public.HasKiloAPIKey = secret.KiloAPIKey != ""
	secret.Public.HasLinearAPIKey = secret.LinearAPIKey != ""

	s.config[id] = secret
	if err := s.saveLocked(); err != nil {
		return runner.Configuration{}, err
	}
	return secret.Public, nil
}

func (s *Memory) GetConfig(id string) (runner.Configuration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfg, ok := s.config[id]
	if !ok {
		return runner.Configuration{}, ErrNotFound
	}
	return cfg.Public, nil
}

func (s *Memory) GetConfigSecret(id string) (runner.ConfigurationSecret, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfg, ok := s.config[id]
	if !ok {
		return runner.ConfigurationSecret{}, ErrNotFound
	}
	return runner.ConfigurationSecret{
		GitHubToken:    cfg.GitHubToken,
		OpenCodeAPIKey: cfg.OpenCodeAPIKey,
		KiloAPIKey:     cfg.KiloAPIKey,
		LinearAPIKey:   cfg.LinearAPIKey,
	}, nil
}

func (s *Memory) ListConfigs() []runner.Configuration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	configs := make([]runner.Configuration, 0, len(s.config))
	for _, cfg := range s.config {
		configs = append(configs, cfg.Public)
	}
	sort.Slice(configs, func(i, j int) bool {
		return configs[i].UpdatedAt.After(configs[j].UpdatedAt)
	})
	return configs
}

func (s *Memory) Create(req runner.RunRequest) (runner.Run, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.runs[req.RequestID]; ok {
		return existing, false, nil
	}

	now := time.Now().UTC()
	run := runner.Run{
		RequestID:      req.RequestID,
		ConfigID:       req.ConfigID,
		RepoURL:        req.RepoURL,
		SourceBranch:   req.SourceBranch,
		Prompt:         req.Prompt,
		WorkDirectory:  defaultString(req.WorkDirectory, "."),
		HarnessRepoURL: req.HarnessRepoURL,
		AgentProvider:  req.AgentProvider,
		AgentModel:     req.AgentModel,
		CreateMR:       req.CreateMR,
		IssueKey:       req.IssueKey,
		LinearIssueKey: req.LinearIssueKey,
		HarnessName:    defaultString(req.HarnessName, "default"),
		SandboxSize:    defaultString(req.SandboxSize, "large"),
		ConfigPath:     req.ConfigPath,
		PushChanges:    req.PushChanges || req.CreateMR,
		Phase:          runner.PhaseQueued,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if req.ConfigID != "" {
		cfg, ok := s.config[req.ConfigID]
		if !ok {
			return runner.Run{}, false, ErrNotFound
		}
		run.ConfigName = cfg.Public.Name
		run.RepoURL = defaultString(run.RepoURL, cfg.Public.RepoURL)
		run.SourceBranch = defaultString(run.SourceBranch, cfg.Public.SourceBranch)
		run.WorkDirectory = defaultString(run.WorkDirectory, cfg.Public.WorkDirectory)
		run.HarnessRepoURL = defaultString(run.HarnessRepoURL, cfg.Public.HarnessRepoURL)
		run.AgentProvider = defaultString(run.AgentProvider, cfg.Public.AgentProvider)
		run.AgentModel = defaultString(run.AgentModel, cfg.Public.AgentModel)
		run.HarnessName = defaultString(run.HarnessName, cfg.Public.HarnessName)
		run.SandboxSize = defaultString(run.SandboxSize, cfg.Public.SandboxSize)
		run.ConfigPath = defaultString(run.ConfigPath, cfg.Public.ConfigPath)
		run.CreateMR = req.CreateMR || cfg.Public.CreateMR
		run.PushChanges = req.PushChanges || run.CreateMR || cfg.Public.PushChanges
	}
	run.AgentProvider = defaultString(run.AgentProvider, "opencode")
	run.AgentModel = defaultAgentModel(run.AgentProvider, run.AgentModel)

	s.runs[run.RequestID] = run
	if err := s.saveLocked(); err != nil {
		return runner.Run{}, false, err
	}
	return run, true, nil
}

func (s *Memory) Get(id string) (runner.Run, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	run, ok := s.runs[id]
	if !ok {
		return runner.Run{}, ErrNotFound
	}
	return run, nil
}

func (s *Memory) Update(id string, mutate func(*runner.Run)) (runner.Run, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.runs[id]
	if !ok {
		return runner.Run{}, ErrNotFound
	}
	mutate(&run)
	run.UpdatedAt = time.Now().UTC()
	s.runs[id] = run
	if err := s.saveLocked(); err != nil {
		return runner.Run{}, err
	}
	return run, nil
}

func (s *Memory) List(limit int) []runner.Run {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return limitRuns(s.runs, limit, "")
}

func (s *Memory) ListByConfig(configID string, limit int) []runner.Run {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return limitRuns(s.runs, limit, configID)
}

func (s *Memory) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var state diskState
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}
	if state.Runs != nil {
		s.runs = state.Runs
	}
	if state.Configurations != nil {
		s.config = state.Configurations
	}
	return nil
}

func (s *Memory) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(diskState{Runs: s.runs, Configurations: s.config}, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func limitRuns(source map[string]runner.Run, limit int, configID string) []runner.Run {
	runs := make([]runner.Run, 0, len(source))
	for _, run := range source {
		if configID == "" || run.ConfigID == configID {
			runs = append(runs, run)
		}
	}
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].CreatedAt.After(runs[j].CreatedAt)
	})
	if limit > 0 && len(runs) > limit {
		return runs[:limit]
	}
	return runs
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func defaultAgentModel(provider, model string) string {
	if model != "" {
		return model
	}
	switch strings.ToLower(provider) {
	case "kilo", "kilocode":
		return "kilo/kilo-auto/free"
	default:
		return "opencode/big-pickle"
	}
}

func slug(value string) string {
	value = strings.ToLower(value)
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}
