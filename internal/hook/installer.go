package hook

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/arvos/hunter/internal/git"
)

const (
	// HunterMarker is used to identify hunter-managed hooks
	HunterMarker = "Hunter: Git blame for AI"
)

var (
	// ErrHookExists indicates a non-hunter hook already exists
	ErrHookExists = errors.New("hook already exists and was not installed by hunter")
)

// Installer manages git hook installation
type Installer struct {
	repo *git.Repo
}

// NewInstaller creates a new hook installer for the given repo
func NewInstaller(repo *git.Repo) *Installer {
	return &Installer{repo: repo}
}

// Install installs the hunter hook
func (i *Installer) Install(hookName string) error {
	hookPath := filepath.Join(i.repo.HooksDir(), hookName)

	// Check if hook already exists
	if content, err := os.ReadFile(hookPath); err == nil {
		// Hook exists - check if it's ours
		if strings.Contains(string(content), HunterMarker) {
			// It's ours, update it
			return i.writeHook(hookPath, hookName)
		}

		// Check if it's a sample hook
		if strings.Contains(string(content), ".sample") {
			return i.writeHook(hookPath, hookName)
		}

		// Existing non-hunter hook - try to chain it
		return i.chainHook(hookPath, hookName, string(content))
	}

	// No existing hook, install fresh
	return i.writeHook(hookPath, hookName)
}

// writeHook writes the hook script to the hooks directory
func (i *Installer) writeHook(hookPath, hookName string) error {
	script := GetHookScript(hookName)
	if script == "" {
		return fmt.Errorf("unknown hook: %s", hookName)
	}

	// Ensure hooks directory exists
	if err := os.MkdirAll(filepath.Dir(hookPath), 0755); err != nil {
		return err
	}

	// Write the hook file
	if err := os.WriteFile(hookPath, []byte(script), 0755); err != nil {
		return err
	}

	return nil
}

// chainHook creates a hook that calls both the existing hook and hunter
func (i *Installer) chainHook(hookPath, hookName, existingContent string) error {
	// Backup the existing hook
	backupPath := hookPath + ".pre-hunter"
	if err := os.WriteFile(backupPath, []byte(existingContent), 0755); err != nil {
		return err
	}

	// Create a chained hook
	chainedScript := fmt.Sprintf(`#!/bin/sh
# %s
# This hook chains the original hook with hunter

# Run the original hook first
if [ -x "%s" ]; then
    "%s" "$@"
    ORIGINAL_EXIT=$?
    if [ $ORIGINAL_EXIT -ne 0 ]; then
        exit $ORIGINAL_EXIT
    fi
fi

# Run hunter hook
%s
`, HunterMarker, backupPath, backupPath, getHunterCommand(hookName))

	return os.WriteFile(hookPath, []byte(chainedScript), 0755)
}

// getHunterCommand returns the hunter command for a hook
func getHunterCommand(hookName string) string {
	switch hookName {
	case "prepare-commit-msg":
		return `hunter hook prepare-commit-msg "$1" 2>/dev/null || true`
	default:
		return ""
	}
}

// Uninstall removes the hunter hook
func (i *Installer) Uninstall(hookName string) error {
	hookPath := filepath.Join(i.repo.HooksDir(), hookName)
	backupPath := hookPath + ".pre-hunter"

	// Check if we have a backup to restore
	if _, err := os.Stat(backupPath); err == nil {
		// Restore the backup
		if err := os.Rename(backupPath, hookPath); err != nil {
			return err
		}
		return nil
	}

	// Check if the current hook is ours
	content, err := os.ReadFile(hookPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Already uninstalled
		}
		return err
	}

	if !strings.Contains(string(content), HunterMarker) {
		return ErrHookExists // Not our hook
	}

	// Remove our hook
	return os.Remove(hookPath)
}

// IsInstalled checks if the hunter hook is installed
func (i *Installer) IsInstalled(hookName string) bool {
	hookPath := filepath.Join(i.repo.HooksDir(), hookName)
	content, err := os.ReadFile(hookPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), HunterMarker)
}

// Status returns the installation status
func (i *Installer) Status(hookName string) string {
	hookPath := filepath.Join(i.repo.HooksDir(), hookName)
	content, err := os.ReadFile(hookPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "not installed"
		}
		return "error: " + err.Error()
	}

	if strings.Contains(string(content), HunterMarker) {
		backupPath := hookPath + ".pre-hunter"
		if _, err := os.Stat(backupPath); err == nil {
			return "installed (chained with existing hook)"
		}
		return "installed"
	}

	return "not installed (other hook present)"
}
