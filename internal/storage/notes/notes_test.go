package notes

import (
	"encoding/json"
	"testing"

	"github.com/arvos/hunter/internal/session"
)

func TestNoteData_JSON(t *testing.T) {
	data := &NoteData{
		Version:      "git-ai/3.0.0",
		SessionID:    "abc123",
		Tool:         "claude",
		Model:        "claude-opus-4-5",
		Plan:         "## Plan\n1. Do something",
		Summary:      "Did something",
		FilesChanged: []string{"file1.go", "file2.go"},
		SessionRef: &SessionRef{
			Path:         "~/.claude/sessions/abc123.jsonl",
			SetupCommand: "hunter restore abc123",
		},
	}

	// Test marshaling
	jsonData, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Test unmarshaling
	var decoded NoteData
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify fields
	if decoded.Version != data.Version {
		t.Errorf("Version = %q, want %q", decoded.Version, data.Version)
	}
	if decoded.SessionID != data.SessionID {
		t.Errorf("SessionID = %q, want %q", decoded.SessionID, data.SessionID)
	}
	if decoded.Tool != data.Tool {
		t.Errorf("Tool = %q, want %q", decoded.Tool, data.Tool)
	}
	if decoded.Model != data.Model {
		t.Errorf("Model = %q, want %q", decoded.Model, data.Model)
	}
	if decoded.Plan != data.Plan {
		t.Errorf("Plan = %q, want %q", decoded.Plan, data.Plan)
	}
	if decoded.Summary != data.Summary {
		t.Errorf("Summary = %q, want %q", decoded.Summary, data.Summary)
	}
	if len(decoded.FilesChanged) != len(data.FilesChanged) {
		t.Errorf("FilesChanged length = %d, want %d", len(decoded.FilesChanged), len(data.FilesChanged))
	}
	if decoded.SessionRef == nil {
		t.Error("SessionRef should not be nil")
	} else {
		if decoded.SessionRef.Path != data.SessionRef.Path {
			t.Errorf("SessionRef.Path = %q, want %q", decoded.SessionRef.Path, data.SessionRef.Path)
		}
		if decoded.SessionRef.SetupCommand != data.SessionRef.SetupCommand {
			t.Errorf("SessionRef.SetupCommand = %q, want %q", decoded.SessionRef.SetupCommand, data.SessionRef.SetupCommand)
		}
	}
}

func TestBuildNoteData(t *testing.T) {
	sess := &session.Session{
		ID:          "abc12345-def6-7890-ghij-klmnopqrstuv",
		Tool:        "claude",
		Model:       "claude-opus-4-5",
		SessionFile: "~/.claude/sessions/abc12345.jsonl",
		PlanFile:    "", // No plan file
	}

	filesChanged := []string{"main.go", "util.go"}
	summary := "Test summary"

	data := BuildNoteData(sess, filesChanged, summary)

	if data.Version != "git-ai/3.0.0" {
		t.Errorf("Version = %q, want %q", data.Version, "git-ai/3.0.0")
	}

	if data.SessionID != sess.ID {
		t.Errorf("SessionID = %q, want %q", data.SessionID, sess.ID)
	}

	if data.Tool != sess.Tool {
		t.Errorf("Tool = %q, want %q", data.Tool, sess.Tool)
	}

	if data.Model != sess.Model {
		t.Errorf("Model = %q, want %q", data.Model, sess.Model)
	}

	if data.Summary != summary {
		t.Errorf("Summary = %q, want %q", data.Summary, summary)
	}

	if len(data.FilesChanged) != len(filesChanged) {
		t.Errorf("FilesChanged length = %d, want %d", len(data.FilesChanged), len(filesChanged))
	}

	if data.SessionRef == nil {
		t.Error("SessionRef should not be nil")
	} else {
		// ShortID should be first 8 chars
		expectedCmd := "hunter restore abc12345"
		if data.SessionRef.SetupCommand != expectedCmd {
			t.Errorf("SetupCommand = %q, want %q", data.SessionRef.SetupCommand, expectedCmd)
		}
	}
}

func TestBuildNoteData_ShortSessionID(t *testing.T) {
	sess := &session.Session{
		ID:          "abc", // Less than 8 characters
		Tool:        "claude",
		SessionFile: "test.jsonl",
	}

	data := BuildNoteData(sess, nil, "")

	// Should handle short IDs gracefully
	if data.SessionRef == nil {
		t.Error("SessionRef should not be nil")
	} else {
		expectedCmd := "hunter restore abc"
		if data.SessionRef.SetupCommand != expectedCmd {
			t.Errorf("SetupCommand = %q, want %q", data.SessionRef.SetupCommand, expectedCmd)
		}
	}
}

func TestNoteData_EmptyOptionalFields(t *testing.T) {
	data := &NoteData{
		Version:   "git-ai/3.0.0",
		SessionID: "abc123",
		Tool:      "claude",
		// All optional fields left empty
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Verify omitempty works
	jsonStr := string(jsonData)
	if contains(jsonStr, "model") {
		t.Error("Empty Model should be omitted from JSON")
	}
	if contains(jsonStr, "plan") {
		t.Error("Empty Plan should be omitted from JSON")
	}
	if contains(jsonStr, "summary") {
		t.Error("Empty Summary should be omitted from JSON")
	}
	if contains(jsonStr, "files_changed") {
		t.Error("Empty FilesChanged should be omitted from JSON")
	}
	if contains(jsonStr, "session_ref") {
		t.Error("Nil SessionRef should be omitted from JSON")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
