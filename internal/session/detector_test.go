package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEncodeProjectPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/Users/arvos/Code/hunter", "-Users-arvos-Code-hunter"},
		{"/home/user/project", "-home-user-project"},
		{"/tmp/test", "-tmp-test"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := encodeProjectPath(tt.input)
			if result != tt.expected {
				t.Errorf("encodeProjectPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFindPlanFile(t *testing.T) {
	// This test checks that FindPlanFile returns empty for non-existent slugs
	result := FindPlanFile("non-existent-slug-12345")
	if result != "" {
		t.Errorf("FindPlanFile for non-existent slug should return empty, got %q", result)
	}
}

func TestParseSessionFile(t *testing.T) {
	// Create a temporary session file
	tmpDir := t.TempDir()
	sessionFile := filepath.Join(tmpDir, "test-session.jsonl")

	content := `{"type":"init","sessionId":"abc123","model":"claude-opus-4-5","cwd":"/test/path","timestamp":"2024-01-01T12:00:00Z"}
{"type":"message","content":"test","timestamp":"2024-01-01T12:01:00Z"}
`
	if err := os.WriteFile(sessionFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	session, err := ParseSessionFile(sessionFile)
	if err != nil {
		t.Fatalf("ParseSessionFile failed: %v", err)
	}

	if session.ID != "abc123" {
		t.Errorf("session.ID = %q, want %q", session.ID, "abc123")
	}

	if session.Model != "claude-opus-4-5" {
		t.Errorf("session.Model = %q, want %q", session.Model, "claude-opus-4-5")
	}

	if session.ProjectPath != "/test/path" {
		t.Errorf("session.ProjectPath = %q, want %q", session.ProjectPath, "/test/path")
	}
}

func TestDetectActiveSession_NoSession(t *testing.T) {
	// Test with a path that doesn't have any sessions
	tmpDir := t.TempDir()

	_, err := DetectActiveSession(tmpDir)
	if err != ErrNoSession {
		t.Errorf("DetectActiveSession should return ErrNoSession for empty dir, got %v", err)
	}
}
