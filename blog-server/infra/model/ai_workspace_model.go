package model

import "time"

type AIWorkspaceModel struct {
	Version int              `json:"version"`
	AdminID uint64           `json:"admin_id"`
	Day     string           `json:"day"`
	Tasks   int              `json:"tasks"`
	Sync    AISyncModel      `json:"sync"`
	Jobs    []AICodeJobModel `json:"jobs"`
}

type AISyncModel struct {
	Phase     string    `json:"phase"`
	Revision  string    `json:"revision"`
	Message   string    `json:"message"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AICodeJobModel struct {
	ID           string    `json:"id"`
	Prompt       string    `json:"prompt"`
	ContextMode  string    `json:"context_mode"`
	Status       string    `json:"status"`
	Context      string    `json:"context"`
	Summary      string    `json:"summary"`
	Diff         string    `json:"diff"`
	Checks       string    `json:"checks"`
	Branch       string    `json:"branch"`
	URL          string    `json:"url"`
	BaseRevision string    `json:"base_revision"`
	ReviewDigest string    `json:"review_digest"`
	TestsPassed  bool      `json:"tests_passed"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	LeaseUntil   time.Time `json:"lease_until"`
	Lease        string    `json:"lease"`
}
