package k8s

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/azhry/nala-chores/internal/runner"
)

type JobOptions struct {
	Namespace  string
	Image      string
	SecretName string
}

type Resources struct {
	RequestCPU    string
	RequestMemory string
	LimitCPU      string
	LimitMemory   string
}

func ResourcesFor(size string) Resources {
	switch strings.ToLower(size) {
	case "small":
		return Resources{"500m", "1Gi", "1", "2Gi"}
	case "large":
		return Resources{"1", "4Gi", "2", "8Gi"}
	case "xlarge":
		return Resources{"2", "8Gi", "4", "16Gi"}
	case "2xlarge":
		return Resources{"4", "16Gi", "8", "32Gi"}
	default:
		return Resources{"1", "4Gi", "2", "8Gi"}
	}
}

func JobName(requestID string) string {
	name := strings.ToLower(requestID)
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '.':
			b.WriteRune('-')
		default:
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		out = "run"
	}
	if len(out) > 55 {
		out = out[:55]
		out = strings.TrimRight(out, "-")
	}
	return "run-" + out
}

func RenderJob(run runner.Run, opts JobOptions) ([]byte, error) {
	if opts.Namespace == "" {
		opts.Namespace = "agent-runner"
	}
	if opts.Image == "" {
		opts.Image = "opencode-runner-backend:local"
	}
	if opts.SecretName == "" {
		opts.SecretName = "runner-secrets"
	}

	data := struct {
		Run       runner.Run
		JobName   string
		Namespace string
		Image     string
		Secret    string
		Resources Resources
	}{
		Run:       run,
		JobName:   JobName(run.RequestID),
		Namespace: opts.Namespace,
		Image:     opts.Image,
		Secret:    opts.SecretName,
		Resources: ResourcesFor(run.SandboxSize),
	}

	tmpl, err := template.New("job").Parse(jobTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse job template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("render job: %w", err)
	}
	return buf.Bytes(), nil
}

const jobTemplate = `apiVersion: batch/v1
kind: Job
metadata:
  name: {{ .JobName }}
  namespace: {{ .Namespace }}
  labels:
    app.kubernetes.io/name: nala-chores
    app.kubernetes.io/component: sandbox
    runner.openai.local/job-name: {{ .JobName | printf "%q" }}
spec:
  ttlSecondsAfterFinished: 3600
  backoffLimit: 0
  template:
    metadata:
      labels:
        app.kubernetes.io/name: nala-chores
        app.kubernetes.io/component: sandbox
        runner.openai.local/job-name: {{ .JobName | printf "%q" }}
    spec:
      restartPolicy: Never
      containers:
        - name: worker
          image: {{ .Image }}
          imagePullPolicy: IfNotPresent
          env:
            - name: REQUEST_ID
              value: {{ .Run.RequestID | printf "%q" }}
            - name: REPO_URL
              value: {{ .Run.RepoURL | printf "%q" }}
            - name: SOURCE_BRANCH
              value: {{ .Run.SourceBranch | printf "%q" }}
            - name: PROMPT
              value: {{ .Run.Prompt | printf "%q" }}
            - name: WORK_DIR
              value: {{ .Run.WorkDirectory | printf "%q" }}
            - name: HARNESS_REPO_URL
              value: {{ .Run.HarnessRepoURL | printf "%q" }}
            - name: HARNESS_NAME
              value: {{ .Run.HarnessName | printf "%q" }}
            - name: SANDBOX_SIZE
              value: {{ .Run.SandboxSize | printf "%q" }}
            - name: CONFIG_PATH
              value: {{ .Run.ConfigPath | printf "%q" }}
            - name: CREATE_MR
              value: {{ .Run.CreateMR | printf "%t" | printf "%q" }}
            - name: PUSH_CHANGES
              value: {{ .Run.PushChanges | printf "%t" | printf "%q" }}
            - name: ISSUE_KEY
              value: {{ .Run.IssueKey | printf "%q" }}
            - name: LINEAR_ISSUE_KEY
              value: {{ .Run.LinearIssueKey | printf "%q" }}
            - name: OPENCODE_DISABLE_AUTOUPDATE
              value: "1"
            - name: OPENCODE_EXPERIMENTAL_BASH_DEFAULT_TIMEOUT_MS
              value: "600000"
            - name: GIT_TOKEN
              valueFrom:
                secretKeyRef:
                  name: {{ .Secret }}
                  key: git_token
                  optional: true
            - name: ANTHROPIC_API_KEY
              valueFrom:
                secretKeyRef:
                  name: {{ .Secret }}
                  key: anthropic_api_key
                  optional: true
            - name: OPENAI_API_KEY
              valueFrom:
                secretKeyRef:
                  name: {{ .Secret }}
                  key: openai_api_key
                  optional: true
            - name: OPENCODE_API_KEY
              valueFrom:
                secretKeyRef:
                  name: {{ .Secret }}
                  key: opencode_api_key
                  optional: true
            - name: LINEAR_API_KEY
              valueFrom:
                secretKeyRef:
                  name: {{ .Secret }}
                  key: linear_api_key
                  optional: true
          resources:
            requests:
              cpu: {{ .Resources.RequestCPU | printf "%q" }}
              memory: {{ .Resources.RequestMemory | printf "%q" }}
            limits:
              cpu: {{ .Resources.LimitCPU | printf "%q" }}
              memory: {{ .Resources.LimitMemory | printf "%q" }}
          volumeMounts:
            - name: workspace
              mountPath: /workspace
      volumes:
        - name: workspace
          emptyDir:
            sizeLimit: 20Gi
`
