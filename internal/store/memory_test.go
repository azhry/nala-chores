package store

import (
	"testing"

	"github.com/azhry/nala-chores/internal/runner"
)

func TestCreateIsIdempotent(t *testing.T) {
	s := NewMemory("")
	req := runner.RunRequest{
		RequestID:    "same",
		RepoURL:      "https://example.com/repo.git",
		SourceBranch: "main",
		Prompt:       "do work",
	}

	first, created, err := s.Create(req)
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("first create should create a run")
	}
	second, created, err := s.Create(req)
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Fatal("second create should be idempotent")
	}
	if first.CreatedAt != second.CreatedAt || second.RequestID != "same" {
		t.Fatalf("unexpected idempotent run: %#v", second)
	}
}

func TestSaveConfigMasksSecrets(t *testing.T) {
	s := NewMemory("")
	cfg, err := s.SaveConfig(runner.ConfigurationInput{
		Name:           "Demo",
		RepoURL:        "https://github.com/example/repo.git",
		HarnessRepoURL: "https://github.com/example/harnesses.git",
		AgentProvider:  "kilocode",
		GitHubToken:    "gh",
		OpenCodeAPIKey: "oc",
		KiloAPIKey:     "kilo",
		LinearAPIKey:   "lin",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.HasGitHubToken || !cfg.HasOpenCodeAPIKey || !cfg.HasKiloAPIKey || !cfg.HasLinearAPIKey {
		t.Fatalf("expected secret presence flags, got %#v", cfg)
	}
	if cfg.AgentProvider != "kilocode" || cfg.AgentModel != "kilo/kilo-auto/free" {
		t.Fatalf("expected Kilo free defaults, got provider=%q model=%q", cfg.AgentProvider, cfg.AgentModel)
	}
	if cfg.HarnessRepoURL != "https://github.com/example/harnesses.git" {
		t.Fatalf("expected harness repo URL to be stored, got %q", cfg.HarnessRepoURL)
	}
	secret, err := s.GetConfigSecret(cfg.ID)
	if err != nil {
		t.Fatal(err)
	}
	if secret.GitHubToken != "gh" || secret.OpenCodeAPIKey != "oc" || secret.KiloAPIKey != "kilo" || secret.LinearAPIKey != "lin" {
		t.Fatalf("unexpected secrets: %#v", secret)
	}
}
