package session

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// jsonlEntry represents a single entry in a Claude session JSONL file
type jsonlEntry struct {
	Type         string          `json:"type"`
	SessionID    string          `json:"sessionId"`
	Slug         string          `json:"slug"`
	Model        string          `json:"model"`
	Cwd          string          `json:"cwd"`
	Message      json.RawMessage `json:"message"`
	Timestamp    string          `json:"timestamp"`
	ParentUUID   string          `json:"parentUuid"`
	UUID         string          `json:"uuid"`
}

// ParseSessionFile parses a Claude session JSONL file and returns session info
func ParseSessionFile(path string) (*Session, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	session := &Session{
		SessionFile:  path,
		Tool:         "claude",
		LastActivity: info.ModTime(),
	}

	scanner := bufio.NewScanner(file)
	// Increase buffer size for long lines (Claude session files can have very long lines)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024) // 10MB max line size

	var sessionEntry jsonlEntry
	foundSessionEntry := false

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var entry jsonlEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // Skip malformed lines
		}

		// Look for the first entry that has session info
		if !foundSessionEntry && entry.SessionID != "" {
			sessionEntry = entry
			foundSessionEntry = true
			break // Found what we need, no need to read the entire file
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Extract session info from session entry
	if sessionEntry.SessionID != "" {
		session.ID = sessionEntry.SessionID
	} else {
		// Use filename as session ID if not in content
		base := filepath.Base(path)
		session.ID = strings.TrimSuffix(base, ".jsonl")
	}

	if sessionEntry.Slug != "" {
		session.Slug = sessionEntry.Slug
	}

	if sessionEntry.Model != "" {
		session.Model = sessionEntry.Model
	}

	if sessionEntry.Cwd != "" {
		session.ProjectPath = sessionEntry.Cwd
	}

	// Parse start time from session entry
	if sessionEntry.Timestamp != "" {
		if t, err := parseTimestamp(sessionEntry.Timestamp); err == nil {
			session.StartTime = t
		}
	}

	// Use file modification time as the primary indicator of last activity
	// Entry timestamps can be stale or missing, but file mod time is always accurate
	// when Claude is actively writing to the session file
	// (session.LastActivity is already set to info.ModTime() above)

	return session, nil
}

// FindPlanFile looks for a plan file associated with a session
func FindPlanFile(slug string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	plansDir := filepath.Join(homeDir, ".claude", "plans")
	planFile := filepath.Join(plansDir, slug+".md")

	if _, err := os.Stat(planFile); err == nil {
		return planFile
	}

	return ""
}

// ReadPlanContent reads the content of a plan file
func ReadPlanContent(planFile string) (string, error) {
	content, err := os.ReadFile(planFile)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// parseTimestamp parses timestamps in various formats Claude uses
func parseTimestamp(ts string) (time.Time, error) {
	// Try RFC3339 with nanoseconds (handles milliseconds too)
	if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
		return t, nil
	}
	// Try standard RFC3339
	if t, err := time.Parse(time.RFC3339, ts); err == nil {
		return t, nil
	}
	// Try without timezone
	if t, err := time.Parse("2006-01-02T15:04:05", ts); err == nil {
		return t, nil
	}
	return time.Time{}, nil
}
