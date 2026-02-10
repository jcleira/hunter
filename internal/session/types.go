package session

import "time"

// Session represents an AI coding session
type Session struct {
	// ID is the unique session identifier (UUID)
	ID string `json:"session_id"`

	// Slug is the human-readable session name (e.g., "tender-snacking-unicorn")
	Slug string `json:"slug,omitempty"`

	// Tool identifies the AI tool (e.g., "claude", "codex", "cursor")
	Tool string `json:"tool"`

	// Model is the AI model used (e.g., "claude-opus-4-5-20251101")
	Model string `json:"model,omitempty"`

	// ProjectPath is the absolute path to the project directory
	ProjectPath string `json:"project_path"`

	// SessionFile is the path to the session JSONL file
	SessionFile string `json:"session_file"`

	// PlanFile is the path to the plan markdown file (if any)
	PlanFile string `json:"plan_file,omitempty"`

	// StartTime is when the session started
	StartTime time.Time `json:"start_time,omitempty"`

	// LastActivity is the last time the session was active
	LastActivity time.Time `json:"last_activity"`

	// IsRunning indicates if the AI process is currently running
	IsRunning bool `json:"is_running"`
}
