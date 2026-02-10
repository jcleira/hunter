package hook

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arvos/hunter/internal/git"
)

func TestGetHookScript(t *testing.T) {
	tests := []struct {
		hookName string
		wantNil  bool
	}{
		{"prepare-commit-msg", false},
		{"unknown-hook", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.hookName, func(t *testing.T) {
			result := GetHookScript(tt.hookName)
			if tt.wantNil && result != "" {
				t.Errorf("GetHookScript(%q) = %q, want empty", tt.hookName, result)
			}
			if !tt.wantNil && result == "" {
				t.Errorf("GetHookScript(%q) = empty, want non-empty", tt.hookName)
			}
		})
	}
}

func TestPrepareCommitMsgScript(t *testing.T) {
	script := PrepareCommitMsgScript

	// Check for required elements
	if !strings.Contains(script, "#!/bin/sh") {
		t.Error("Script should start with shebang")
	}

	if !strings.Contains(script, HunterMarker) {
		t.Error("Script should contain HunterMarker")
	}

	if !strings.Contains(script, "hunter hook prepare-commit-msg") {
		t.Error("Script should call hunter hook command")
	}

	// Check that merge/squash commits are skipped
	if !strings.Contains(script, "merge") || !strings.Contains(script, "squash") {
		t.Error("Script should skip merge and squash commits")
	}
}

func TestInstaller_InstallAndUninstall(t *testing.T) {
	// Create a temporary git repository
	tmpDir := t.TempDir()

	// Initialize git repo
	if err := os.MkdirAll(filepath.Join(tmpDir, ".git", "hooks"), 0755); err != nil {
		t.Fatal(err)
	}

	repo := &git.Repo{
		Path:   tmpDir,
		GitDir: filepath.Join(tmpDir, ".git"),
	}

	installer := NewInstaller(repo)
	hookName := "prepare-commit-msg"

	// Test initial status
	if installer.IsInstalled(hookName) {
		t.Error("Hook should not be installed initially")
	}

	status := installer.Status(hookName)
	if status != "not installed" {
		t.Errorf("Status = %q, want %q", status, "not installed")
	}

	// Test install
	if err := installer.Install(hookName); err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	if !installer.IsInstalled(hookName) {
		t.Error("Hook should be installed after Install()")
	}

	status = installer.Status(hookName)
	if status != "installed" {
		t.Errorf("Status = %q, want %q", status, "installed")
	}

	// Verify hook file content
	hookPath := filepath.Join(repo.HooksDir(), hookName)
	content, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("Failed to read hook file: %v", err)
	}

	if !strings.Contains(string(content), HunterMarker) {
		t.Error("Hook file should contain HunterMarker")
	}

	// Verify file is executable
	info, err := os.Stat(hookPath)
	if err != nil {
		t.Fatalf("Failed to stat hook file: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Error("Hook file should be executable")
	}

	// Test reinstall (should be idempotent)
	if err := installer.Install(hookName); err != nil {
		t.Fatalf("Reinstall failed: %v", err)
	}

	// Test uninstall
	if err := installer.Uninstall(hookName); err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}

	if installer.IsInstalled(hookName) {
		t.Error("Hook should not be installed after Uninstall()")
	}

	// Verify hook file is removed
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Error("Hook file should be removed after uninstall")
	}
}

func TestInstaller_ChainExistingHook(t *testing.T) {
	// Create a temporary git repository
	tmpDir := t.TempDir()

	hooksDir := filepath.Join(tmpDir, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		t.Fatal(err)
	}

	repo := &git.Repo{
		Path:   tmpDir,
		GitDir: filepath.Join(tmpDir, ".git"),
	}

	hookName := "prepare-commit-msg"
	hookPath := filepath.Join(hooksDir, hookName)

	// Create an existing hook
	existingContent := "#!/bin/sh\necho 'existing hook'"
	if err := os.WriteFile(hookPath, []byte(existingContent), 0755); err != nil {
		t.Fatal(err)
	}

	installer := NewInstaller(repo)

	// Install should chain with existing hook
	if err := installer.Install(hookName); err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	// Check that original was backed up
	backupPath := hookPath + ".pre-hunter"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Error("Original hook should be backed up")
	}

	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("Failed to read backup: %v", err)
	}
	if string(backupContent) != existingContent {
		t.Error("Backup content should match original")
	}

	// Check status reflects chaining
	status := installer.Status(hookName)
	if !strings.Contains(status, "chained") {
		t.Errorf("Status = %q, should mention chaining", status)
	}

	// Uninstall should restore original
	if err := installer.Uninstall(hookName); err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}

	restoredContent, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("Failed to read restored hook: %v", err)
	}
	if string(restoredContent) != existingContent {
		t.Error("Original hook should be restored after uninstall")
	}
}
