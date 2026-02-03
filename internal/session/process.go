package session

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// ProcessInfo contains information about a running process
type ProcessInfo struct {
	PID     int
	Command string
	Cwd     string
}

// FindClaudeProcesses finds running Claude Code processes
func FindClaudeProcesses() ([]ProcessInfo, error) {
	// Use ps to find Claude-related processes
	// Look for node processes running claude
	cmd := exec.Command("ps", "-eo", "pid,comm,args")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var processes []ProcessInfo
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		// Skip header
		if fields[0] == "PID" {
			continue
		}

		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}

		// Check if this is a Claude-related process
		args := strings.Join(fields[2:], " ")
		if !isClaudeProcess(args) {
			continue
		}

		// Get the working directory of the process
		cwd := getProcessCwd(pid)

		processes = append(processes, ProcessInfo{
			PID:     pid,
			Command: fields[1],
			Cwd:     cwd,
		})
	}

	return processes, nil
}

// isClaudeProcess checks if the process arguments indicate a Claude process
func isClaudeProcess(args string) bool {
	claudeIndicators := []string{
		"claude",
		"@anthropic/claude-code",
		".claude",
	}

	argsLower := strings.ToLower(args)
	for _, indicator := range claudeIndicators {
		if strings.Contains(argsLower, indicator) {
			return true
		}
	}
	return false
}

// getProcessCwd gets the current working directory of a process
func getProcessCwd(pid int) string {
	// On macOS, use lsof to find the cwd
	cmd := exec.Command("lsof", "-p", strconv.Itoa(pid), "-Fn", "-a", "-d", "cwd")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "n") {
			return strings.TrimPrefix(line, "n")
		}
	}

	return ""
}

// FindClaudeProcessForProject finds a running Claude process for a specific project
func FindClaudeProcessForProject(projectPath string) *ProcessInfo {
	processes, err := FindClaudeProcesses()
	if err != nil {
		return nil
	}

	// Normalize the project path
	projectPath = normalizePath(projectPath)

	for _, proc := range processes {
		procCwd := normalizePath(proc.Cwd)
		if procCwd == projectPath || strings.HasPrefix(procCwd, projectPath+string(os.PathSeparator)) {
			return &proc
		}
	}

	return nil
}

// normalizePath normalizes a path for comparison
func normalizePath(path string) string {
	// Resolve symlinks and clean the path
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		resolved = path
	}
	return filepath.Clean(resolved)
}
