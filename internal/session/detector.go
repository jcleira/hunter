package session

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	// SessionTimeout is the maximum time since last activity for a session to be considered active
	SessionTimeout = 15 * time.Minute
)

var (
	// ErrNoSession indicates no active session was found
	ErrNoSession = errors.New("no active AI session found")
)

// DetectActiveSession detects an active AI session for the given project path
func DetectActiveSession(projectPath string) (*Session, error) {
	// Normalize the project path
	projectPath = normalizePath(projectPath)

	// Strategy 1: Check for running Claude process with matching cwd
	if session := detectFromRunningProcess(projectPath); session != nil {
		return session, nil
	}

	// Strategy 2: Check for recently modified session files
	if session := detectFromRecentSessionFile(projectPath); session != nil {
		return session, nil
	}

	return nil, ErrNoSession
}

// detectFromRunningProcess checks if there's a running Claude process for this project
func detectFromRunningProcess(projectPath string) *Session {
	proc := FindClaudeProcessForProject(projectPath)
	if proc == nil {
		return nil
	}

	// Found a running process, now find the corresponding session file
	session := findSessionFileForProject(projectPath)
	if session != nil {
		session.IsRunning = true
	}
	return session
}

// detectFromRecentSessionFile checks for recently modified session files
func detectFromRecentSessionFile(projectPath string) *Session {
	session := findSessionFileForProject(projectPath)
	if session == nil {
		return nil
	}

	// Check if the session was recently active
	if time.Since(session.LastActivity) > SessionTimeout {
		return nil
	}

	return session
}

// findSessionFileForProject finds the most recent session file for a project
func findSessionFileForProject(projectPath string) *Session {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	// Claude stores sessions in ~/.claude/projects/[encoded-path]/
	encodedPath := encodeProjectPath(projectPath)
	sessionsDir := filepath.Join(homeDir, ".claude", "projects", encodedPath)

	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return nil
	}

	// Collect all session files with their modification times
	type sessionFile struct {
		path    string
		modTime time.Time
	}
	var sessionFiles []sessionFile

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}

		path := filepath.Join(sessionsDir, entry.Name())
		info, err := os.Stat(path)
		if err != nil {
			continue
		}

		sessionFiles = append(sessionFiles, sessionFile{
			path:    path,
			modTime: info.ModTime(),
		})
	}

	if len(sessionFiles) == 0 {
		return nil
	}

	// Sort by modification time, most recent first
	sort.Slice(sessionFiles, func(i, j int) bool {
		return sessionFiles[i].modTime.After(sessionFiles[j].modTime)
	})

	// Parse the most recent session file
	session, err := ParseSessionFile(sessionFiles[0].path)
	if err != nil {
		return nil
	}

	session.ProjectPath = projectPath

	// Try to find a matching plan file
	if session.Slug != "" {
		session.PlanFile = FindPlanFile(session.Slug)
	}

	return session
}

// encodeProjectPath encodes a project path the same way Claude does
// /Users/arvos/Code/hunter -> -Users-arvos-Code-hunter
func encodeProjectPath(path string) string {
	// Replace path separators with dashes
	encoded := strings.ReplaceAll(path, string(os.PathSeparator), "-")
	// Remove leading dash if present (from absolute path starting with /)
	encoded = strings.TrimPrefix(encoded, "-")
	// Add leading dash back (Claude's format)
	return "-" + encoded
}

// FindAllSessions finds all sessions for a project, including inactive ones
func FindAllSessions(projectPath string) ([]*Session, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	projectPath = normalizePath(projectPath)
	encodedPath := encodeProjectPath(projectPath)
	sessionsDir := filepath.Join(homeDir, ".claude", "projects", encodedPath)

	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var sessions []*Session

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}

		path := filepath.Join(sessionsDir, entry.Name())
		session, err := ParseSessionFile(path)
		if err != nil {
			continue
		}

		session.ProjectPath = projectPath
		if session.Slug != "" {
			session.PlanFile = FindPlanFile(session.Slug)
		}

		sessions = append(sessions, session)
	}

	// Sort by last activity, most recent first
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].LastActivity.After(sessions[j].LastActivity)
	})

	return sessions, nil
}

// FindSessionByID finds a specific session by its ID
func FindSessionByID(projectPath, sessionID string) (*Session, error) {
	sessions, err := FindAllSessions(projectPath)
	if err != nil {
		return nil, err
	}

	// Support partial session ID matching
	for _, session := range sessions {
		if session.ID == sessionID || strings.HasPrefix(session.ID, sessionID) {
			return session, nil
		}
	}

	return nil, ErrNoSession
}
