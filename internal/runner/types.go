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
	RequestID      string `json:"request_id"`
	ConfigID       string `json:"config_id"`
	RepoURL        string `json:"repo_url"`
	SourceBranch   string `json:"source_branch"`
	Prompt         string `json:"prompt"`
	WorkDirectory  string `json:"work_directory"`
	HarnessRepoURL string `json:"harness_repo_url"`
	AgentProvider  string `json:"agent_provider"`
	AgentModel     string `json:"agent_model"`
	CreateMR       bool   `json:"create_mr"`
	IssueKey       string `json:"issue_key"`
	LinearIssueKey string `json:"linear_issue_key"`
	HarnessName    string `json:"harness_name"`
	SandboxSize    string `json:"sandbox_size"`
	ConfigPath     string `json:"config_path"`
	PushChanges    bool   `json:"push_changes"`
}

type Run struct {
	RequestID      string     `json:"request_id"`
	ConfigID       string     `json:"config_id,omitempty"`
	ConfigName     string     `json:"config_name,omitempty"`
	RepoURL        string     `json:"repo_url"`
	SourceBranch   string     `json:"source_branch"`
	Prompt         string     `json:"prompt,omitempty"`
	WorkDirectory  string     `json:"work_directory"`
	HarnessRepoURL string     `json:"harness_repo_url,omitempty"`
	AgentProvider  string     `json:"agent_provider"`
	AgentModel     string     `json:"agent_model"`
	CreateMR       bool       `json:"create_mr"`
	IssueKey       string     `json:"issue_key,omitempty"`
	LinearIssueKey string     `json:"linear_issue_key,omitempty"`
	HarnessName    string     `json:"harness_name"`
	SandboxSize    string     `json:"sandbox_size"`
	ConfigPath     string     `json:"config_path,omitempty"`
	PushChanges    bool       `json:"push_changes"`
	Phase          Phase      `json:"phase"`
	Message        string     `json:"message,omitempty"`
	JobName        string     `json:"job_name,omitempty"`
	LogsCommand    string     `json:"logs_command,omitempty"`
	MRURL          string     `json:"mr_url,omitempty"`
	ExitCode       *int       `json:"exit_code,omitempty"`
	Logs           string     `json:"logs,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

type RunList struct {
	Runs []Run `json:"runs"`
}

type ConfigurationInput struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	RepoURL             string `json:"repo_url"`
	SourceBranch        string `json:"source_branch"`
	WorkDirectory       string `json:"work_directory"`
	HarnessRepoURL      string `json:"harness_repo_url"`
	AgentProvider       string `json:"agent_provider"`
	AgentModel          string `json:"agent_model"`
	HarnessName         string `json:"harness_name"`
	SandboxSize         string `json:"sandbox_size"`
	ConfigPath          string `json:"config_path"`
	CreateMR            bool   `json:"create_mr"`
	PushChanges         bool   `json:"push_changes"`
	GitHubToken         string `json:"github_token"`
	OpenCodeAPIKey      string `json:"opencode_api_key"`
	KiloAPIKey          string `json:"kilo_api_key"`
	LinearAPIKey        string `json:"linear_api_key"`
	ClearGitHubToken    bool   `json:"clear_github_token"`
	ClearOpenCodeAPIKey bool   `json:"clear_opencode_api_key"`
	ClearKiloAPIKey     bool   `json:"clear_kilo_api_key"`
	ClearLinearAPIKey   bool   `json:"clear_linear_api_key"`
}

type Configuration struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	RepoURL           string    `json:"repo_url"`
	SourceBranch      string    `json:"source_branch"`
	WorkDirectory     string    `json:"work_directory"`
	HarnessRepoURL    string    `json:"harness_repo_url,omitempty"`
	AgentProvider     string    `json:"agent_provider"`
	AgentModel        string    `json:"agent_model"`
	HarnessName       string    `json:"harness_name"`
	SandboxSize       string    `json:"sandbox_size"`
	ConfigPath        string    `json:"config_path,omitempty"`
	CreateMR          bool      `json:"create_mr"`
	PushChanges       bool      `json:"push_changes"`
	HasGitHubToken    bool      `json:"has_github_token"`
	HasOpenCodeAPIKey bool      `json:"has_opencode_api_key"`
	HasKiloAPIKey     bool      `json:"has_kilo_api_key"`
	HasLinearAPIKey   bool      `json:"has_linear_api_key"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type ConfigurationSecret struct {
	GitHubToken    string
	OpenCodeAPIKey string
	KiloAPIKey     string
	LinearAPIKey   string
}

type ConfigurationList struct {
	Configurations []Configuration `json:"configurations"`
}
