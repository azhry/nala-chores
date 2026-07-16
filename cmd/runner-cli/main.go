package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/azhry/nala-chores/internal/runner"
)

const defaultAPIURL = "http://127.0.0.1:8080"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "submit":
		err = submit(os.Args[2:])
	case "status":
		err = status(os.Args[2:])
	case "list":
		err = list(os.Args[2:])
	case "stop":
		err = stop(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func submit(args []string) error {
	fs := flag.NewFlagSet("submit", flag.ExitOnError)
	apiURL := fs.String("api", env("RUNNER_API_URL", defaultAPIURL), "runner-manager base URL")
	requestID := fs.String("request-id", "", "idempotency key; generated when empty")
	repoURL := fs.String("repo", "", "git repo URL; defaults to origin remote")
	branch := fs.String("branch", "", "source branch; defaults to current branch")
	prompt := fs.String("prompt", "", "task prompt")
	promptFile := fs.String("prompt-file", "", "file containing task prompt")
	workdir := fs.String("workdir", ".", "repo subdirectory for OpenCode")
	harness := fs.String("harness", "default", "harness name from .opencode-runner.yml")
	size := fs.String("sandbox-size", "large", "sandbox size: small, large, xlarge, 2xlarge")
	issue := fs.String("issue", "", "issue key to pass to the harness")
	configPath := fs.String("config", "", "optional repo-relative harness config path")
	createMR := fs.Bool("mr", false, "create a merge request after validation")
	noMR := fs.Bool("no-mr", false, "do not create a merge request")
	pushChanges := fs.Bool("push", false, "commit and push changes without creating an MR")
	strictDirty := fs.Bool("strict-dirty", false, "fail when the local working tree has uncommitted changes")
	pushBranch := fs.Bool("push-branch", false, "push the current branch before submitting")
	wait := fs.Bool("wait", false, "poll until the run leaves active phases")
	_ = fs.Parse(args)

	if *promptFile != "" {
		data, err := os.ReadFile(*promptFile)
		if err != nil {
			return err
		}
		*prompt = strings.TrimSpace(string(data))
	}
	if *prompt == "" {
		return errors.New("--prompt or --prompt-file is required")
	}
	if *noMR {
		*createMR = false
	}

	if *repoURL == "" {
		value, err := git("config", "--get", "remote.origin.url")
		if err != nil {
			return fmt.Errorf("detect repo URL: %w", err)
		}
		*repoURL = strings.TrimSpace(value)
	}
	if *branch == "" {
		value, err := git("branch", "--show-current")
		if err != nil {
			return fmt.Errorf("detect current branch: %w", err)
		}
		*branch = strings.TrimSpace(value)
	}
	if *branch == "" {
		return errors.New("could not determine current branch; pass --branch")
	}
	if *strictDirty {
		dirty, err := git("status", "--porcelain")
		if err != nil {
			return fmt.Errorf("check dirty tree: %w", err)
		}
		if strings.TrimSpace(dirty) != "" {
			return errors.New("working tree has uncommitted changes")
		}
	}
	if *pushBranch || *createMR {
		if _, err := git("push", "-u", "origin", *branch); err != nil {
			return fmt.Errorf("push branch: %w", err)
		}
	}
	if *requestID == "" {
		*requestID = makeRequestID(*repoURL, *branch)
	}

	req := runner.RunRequest{
		RequestID:     *requestID,
		RepoURL:       *repoURL,
		SourceBranch:  *branch,
		Prompt:        *prompt,
		WorkDirectory: *workdir,
		CreateMR:      *createMR,
		IssueKey:      *issue,
		HarnessName:   *harness,
		SandboxSize:   *size,
		ConfigPath:    *configPath,
		PushChanges:   *pushChanges,
	}
	var run runner.Run
	if err := postJSON(*apiURL+"/runs", req, &run); err != nil {
		return err
	}
	printRun(run)
	if *wait {
		return waitForRun(*apiURL, run.RequestID)
	}
	return nil
}

func status(args []string) error {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	apiURL := fs.String("api", env("RUNNER_API_URL", defaultAPIURL), "runner-manager base URL")
	last := fs.Bool("last", false, "show most recent run")
	_ = fs.Parse(args)

	if *last {
		var runs runner.RunList
		if err := getJSON(*apiURL+"/runs?limit=1", &runs); err != nil {
			return err
		}
		if len(runs.Runs) == 0 {
			return errors.New("no runs found")
		}
		printRun(runs.Runs[0])
		return nil
	}

	if fs.NArg() != 1 {
		return errors.New("status requires <request_id> or --last")
	}
	var run runner.Run
	if err := getJSON(*apiURL+"/runs/"+fs.Arg(0), &run); err != nil {
		return err
	}
	printRun(run)
	return nil
}

func list(args []string) error {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	apiURL := fs.String("api", env("RUNNER_API_URL", defaultAPIURL), "runner-manager base URL")
	limit := fs.Int("limit", 20, "number of runs")
	_ = fs.Parse(args)

	var runs runner.RunList
	if err := getJSON(fmt.Sprintf("%s/runs?limit=%d", *apiURL, *limit), &runs); err != nil {
		return err
	}
	for _, run := range runs.Runs {
		fmt.Printf("%s\t%s\t%s\t%s\n", run.RequestID, run.Phase, run.SourceBranch, run.UpdatedAt.Format(time.RFC3339))
	}
	return nil
}

func stop(args []string) error {
	fs := flag.NewFlagSet("stop", flag.ExitOnError)
	apiURL := fs.String("api", env("RUNNER_API_URL", defaultAPIURL), "runner-manager base URL")
	_ = fs.Parse(args)
	if fs.NArg() != 1 {
		return errors.New("stop requires <request_id>")
	}
	var run runner.Run
	if err := postJSON(*apiURL+"/runs/"+fs.Arg(0)+"/stop", map[string]string{}, &run); err != nil {
		return err
	}
	printRun(run)
	return nil
}

func waitForRun(apiURL, requestID string) error {
	for {
		var run runner.Run
		if err := getJSON(apiURL+"/runs/"+requestID, &run); err != nil {
			return err
		}
		printRun(run)
		switch run.Phase {
		case runner.PhaseSucceeded, runner.PhaseFailed, runner.PhaseCancelled:
			return nil
		}
		time.Sleep(5 * time.Second)
	}
}

func printRun(run runner.Run) {
	fmt.Printf("request_id: %s\nphase: %s\nbranch: %s\njob: %s\nlogs: %s\nmessage: %s\n",
		run.RequestID, run.Phase, run.SourceBranch, run.JobName, run.LogsCommand, run.Message)
	if run.MRURL != "" {
		fmt.Println("mr_url:", run.MRURL)
	}
}

func getJSON(url string, out any) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decodeResponse(resp, out)
}

func postJSON(url string, in, out any) error {
	body, err := json.Marshal(in)
	if err != nil {
		return err
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decodeResponse(resp, out)
}

func decodeResponse(resp *http.Response, out any) error {
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(data)))
	}
	return json.Unmarshal(data, out)
}

func git(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %s", strings.Join(args, " "), strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func makeRequestID(repoURL, branch string) string {
	base := strings.TrimSuffix(filepath.Base(repoURL), ".git")
	raw := fmt.Sprintf("usr-%s-%s-%d", base, branch, time.Now().Unix())
	raw = strings.ToLower(raw)
	var b strings.Builder
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func usage() {
	fmt.Fprintf(os.Stderr, `runner-cli submits OpenCode sandbox runs.

Usage:
  runner-cli submit --prompt "Implement the change" [flags]
  runner-cli status <request_id>
  runner-cli status --last
  runner-cli list
  runner-cli stop <request_id>
`)
}
