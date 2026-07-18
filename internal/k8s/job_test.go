package k8s

import (
	"strings"
	"testing"

	"github.com/azhry/nala-chores/internal/runner"
)

func TestJobNameSanitizesRequestID(t *testing.T) {
	got := JobName("User/Repo Feature_X")
	if got != "run-user-repo-feature-x" {
		t.Fatalf("JobName() = %q", got)
	}
}

func TestRenderJobIncludesCoreEnvironment(t *testing.T) {
	run := runner.Run{
		RequestID:      "req-1",
		RepoURL:        "https://github.com/example/repo.git",
		SourceBranch:   "feature/demo",
		Prompt:         "change it",
		WorkDirectory:  "go",
		HarnessRepoURL: "https://github.com/example/harnesses.git",
		HarnessName:    "default",
		SandboxSize:    "xlarge",
	}

	manifest, err := RenderJob(run, JobOptions{Namespace: "agent-runner"})
	if err != nil {
		t.Fatal(err)
	}
	text := string(manifest)
	for _, want := range []string{
		"name: run-req-1",
		"value: \"https://github.com/example/repo.git\"",
		"value: \"https://github.com/example/harnesses.git\"",
		"value: \"feature/demo\"",
		"memory: \"16Gi\"",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered manifest missing %q:\n%s", want, text)
		}
	}
}
