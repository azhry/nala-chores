package store

import (
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/azhry/nala-chores/internal/runner"
)

var ErrNotFound = errors.New("run not found")

type Memory struct {
	mu   sync.RWMutex
	runs map[string]runner.Run
}

func NewMemory() *Memory {
	return &Memory{runs: map[string]runner.Run{}}
}

func (s *Memory) Create(req runner.RunRequest) (runner.Run, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.runs[req.RequestID]; ok {
		return existing, false
	}

	now := time.Now().UTC()
	run := runner.Run{
		RequestID:     req.RequestID,
		RepoURL:       req.RepoURL,
		SourceBranch:  req.SourceBranch,
		Prompt:        req.Prompt,
		WorkDirectory: defaultString(req.WorkDirectory, "."),
		CreateMR:      req.CreateMR,
		IssueKey:      req.IssueKey,
		HarnessName:   defaultString(req.HarnessName, "default"),
		SandboxSize:   defaultString(req.SandboxSize, "large"),
		ConfigPath:    req.ConfigPath,
		PushChanges:   req.PushChanges || req.CreateMR,
		Phase:         runner.PhaseQueued,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	s.runs[run.RequestID] = run
	return run, true
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
	return run, nil
}

func (s *Memory) List(limit int) []runner.Run {
	s.mu.RLock()
	defer s.mu.RUnlock()

	runs := make([]runner.Run, 0, len(s.runs))
	for _, run := range s.runs {
		runs = append(runs, run)
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
