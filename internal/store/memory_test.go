package store

import (
	"testing"

	"github.com/azhry/nala-chores/internal/runner"
)

func TestCreateIsIdempotent(t *testing.T) {
	s := NewMemory()
	req := runner.RunRequest{
		RequestID:    "same",
		RepoURL:      "https://example.com/repo.git",
		SourceBranch: "main",
		Prompt:       "do work",
	}

	first, created := s.Create(req)
	if !created {
		t.Fatal("first create should create a run")
	}
	second, created := s.Create(req)
	if created {
		t.Fatal("second create should be idempotent")
	}
	if first.CreatedAt != second.CreatedAt || second.RequestID != "same" {
		t.Fatalf("unexpected idempotent run: %#v", second)
	}
}
