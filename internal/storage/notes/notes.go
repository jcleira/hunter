package notes

import (
	"encoding/json"

	"github.com/arvos/hunter/internal/git"
	"github.com/arvos/hunter/internal/session"
)

const (
	// NotesRef is the git notes reference for AI session data
	NotesRef = "ai"
)

// NoteData represents the data stored in git notes for a commit
type NoteData struct {
	// Format version for compatibility
	Version string `json:"version"`

	// SessionID is the unique session identifier
	SessionID string `json:"session_id"`

	// Tool identifies the AI tool (e.g., "claude", "codex")
	Tool string `json:"tool"`

	// Model is the AI model used
	Model string `json:"model,omitempty"`

	// Plan is the content of the plan file
	Plan string `json:"plan,omitempty"`

	// Summary is an LLM-generated summary of what was implemented
	Summary string `json:"summary,omitempty"`

	// FilesChanged lists the files that were changed
	FilesChanged []string `json:"files_changed,omitempty"`

	// SessionRef contains info for restoring the session locally
	SessionRef *SessionRef `json:"session_ref,omitempty"`
}

// SessionRef contains information for restoring a session locally
type SessionRef struct {
	// Path to the session file
	Path string `json:"path"`

	// SetupCommand to recreate session context
	SetupCommand string `json:"setup_command"`
}

// Store handles reading and writing AI session notes
type Store struct {
	repo *git.Repo
}

// NewStore creates a new notes store for the given repository
func NewStore(repo *git.Repo) *Store {
	return &Store{repo: repo}
}

// Get retrieves the note data for a commit
func (s *Store) Get(commit string) (*NoteData, error) {
	content, err := s.repo.GetNote(commit, NotesRef)
	if err != nil {
		return nil, err
	}

	if content == "" {
		return nil, nil
	}

	var data NoteData
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return nil, err
	}

	return &data, nil
}

// Set stores note data for a commit
func (s *Store) Set(commit string, data *NoteData) error {
	// Ensure version is set
	if data.Version == "" {
		data.Version = "git-ai/3.0.0"
	}

	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return s.repo.SetNote(commit, NotesRef, string(content))
}

// Push pushes notes to a remote
func (s *Store) Push(remote string) error {
	return s.repo.PushNotes(remote, NotesRef)
}

// Fetch fetches notes from a remote
func (s *Store) Fetch(remote string) error {
	return s.repo.FetchNotes(remote, NotesRef)
}

// BuildNoteData creates note data from a session
func BuildNoteData(sess *session.Session, filesChanged []string, summary string) *NoteData {
	data := &NoteData{
		Version:      "git-ai/3.0.0",
		SessionID:    sess.ID,
		Tool:         sess.Tool,
		Model:        sess.Model,
		FilesChanged: filesChanged,
		Summary:      summary,
	}

	// Add plan content if available
	if sess.PlanFile != "" {
		if content, err := session.ReadPlanContent(sess.PlanFile); err == nil {
			data.Plan = content
		}
	}

	// Add session ref
	shortID := sess.ID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}

	data.SessionRef = &SessionRef{
		Path:         sess.SessionFile,
		SetupCommand: "hunter restore " + shortID,
	}

	return data
}
