package api

import (
	"testing"
	"time"

	"github.com/azhry/nala-chores/internal/runner"
)

func TestApplyRunnerResultExtractsFinalMRURL(t *testing.T) {
	logs := `{"type":"tool_use","part":{"state":{"output":"{\"nested\":true}"}}}
[2026-07-18T10:36:41Z] creating merge request
{
  "request_id": "nala-grow-123",
  "status": "succeeded",
  "message": "run completed",
  "mr_url": "https://github.com/azhry/nala-grow/pull/53",
  "completed_at": "2026-07-18T10:36:42Z"
}`

	run := runner.Run{
		RequestID: "nala-grow-123",
		Phase:     runner.PhaseSucceeded,
		Message:   "Reached expected number of succeeded pods",
	}

	applyRunnerResult(&run, logs)

	if run.MRURL != "https://github.com/azhry/nala-grow/pull/53" {
		t.Fatalf("MRURL = %q", run.MRURL)
	}
	if run.Message != "run completed" {
		t.Fatalf("Message = %q", run.Message)
	}
	if run.CompletedAt == nil {
		t.Fatal("CompletedAt was nil")
	}
	if want := time.Date(2026, 7, 18, 10, 36, 42, 0, time.UTC); !run.CompletedAt.Equal(want) {
		t.Fatalf("CompletedAt = %s, want %s", run.CompletedAt, want)
	}
}

func TestExtractRunnerResultUsesLatestResult(t *testing.T) {
	logs := `{"request_id":"nala-grow-123","status":"failed","message":"old","completed_at":"2026-07-18T10:30:00Z"}
{"request_id":"nala-grow-123","status":"succeeded","message":"run completed","mr_url":"https://github.com/azhry/nala-grow/pull/53","completed_at":"2026-07-18T10:36:42Z"}`

	result, ok := extractRunnerResult("nala-grow-123", logs)
	if !ok {
		t.Fatal("expected result")
	}
	if result.Status != "succeeded" || result.MRURL == "" {
		t.Fatalf("unexpected result: %+v", result)
	}
}
