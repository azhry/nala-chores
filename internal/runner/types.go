package runner

import "time"

type Phase string

const (
	PhaseQueued       Phase = "QUEUED"
	PhaseProvisioning Phase = "PROVISIONING"
	PhaseSetup        Phase = "SETUP"
	PhasePlanning     Phase = "PLANNING"
	PhaseEditing      Phase = "EDITING"
	PhaseValidating   Phase = "VALIDATING"
	PhaseFixing       Phase = "FIXING"
	PhaseCommitting   Phase = "COMMITTING"
	PhasePushing      Phase = "PUSHING"
	PhaseMRCreating   Phase = "MR_CREATING"
	PhaseSucceeded    Phase = "SUCCEEDED"
	PhaseFailed       Phase = "FAILED"
	PhaseCancelled    Phase = "CANCELLED"
)

type RunRequest struct {
	RequestID     string `json:"request_id"`
	RepoURL       string `json:"repo_url"`
	SourceBranch  string `json:"source_branch"`
	Prompt        string `json:"prompt"`
	WorkDirectory string `json:"work_directory"`
	CreateMR      bool   `json:"create_mr"`
	IssueKey      string `json:"issue_key"`
	HarnessName   string `json:"harness_name"`
	SandboxSize   string `json:"sandbox_size"`
	ConfigPath    string `json:"config_path"`
	PushChanges   bool   `json:"push_changes"`
}

type Run struct {
	RequestID     string     `json:"request_id"`
	RepoURL       string     `json:"repo_url"`
	SourceBranch  string     `json:"source_branch"`
	Prompt        string     `json:"prompt,omitempty"`
	WorkDirectory string     `json:"work_directory"`
	CreateMR      bool       `json:"create_mr"`
	IssueKey      string     `json:"issue_key,omitempty"`
	HarnessName   string     `json:"harness_name"`
	SandboxSize   string     `json:"sandbox_size"`
	ConfigPath    string     `json:"config_path,omitempty"`
	PushChanges   bool       `json:"push_changes"`
	Phase         Phase      `json:"phase"`
	Message       string     `json:"message,omitempty"`
	JobName       string     `json:"job_name,omitempty"`
	LogsCommand   string     `json:"logs_command,omitempty"`
	MRURL         string     `json:"mr_url,omitempty"`
	ExitCode      *int       `json:"exit_code,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

type RunList struct {
	Runs []Run `json:"runs"`
}
