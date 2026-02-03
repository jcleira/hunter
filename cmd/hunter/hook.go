package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/arvos/hunter/internal/git"
	"github.com/arvos/hunter/internal/session"
	"github.com/spf13/cobra"
)

var hookCmd = &cobra.Command{
	Use:    "hook",
	Short:  "Git hook commands (internal use)",
	Hidden: true, // Hide from main help - these are called by git hooks
}

var hookPrepareCommitMsgCmd = &cobra.Command{
	Use:   "prepare-commit-msg <msg-file>",
	Short: "Called by git prepare-commit-msg hook",
	Args:  cobra.ExactArgs(1),
	RunE:  runHookPrepareCommitMsg,
}

func init() {
	rootCmd.AddCommand(hookCmd)
	hookCmd.AddCommand(hookPrepareCommitMsgCmd)
}

func runHookPrepareCommitMsg(cmd *cobra.Command, args []string) error {
	msgFile := args[0]

	// Get the repository root
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// Detect active session
	sess, err := session.DetectActiveSession(cwd)
	if err != nil {
		// No active session - this is fine, just exit silently
		return nil
	}

	// Build trailers
	trailers := &git.Trailers{
		SessionID: sess.ID,
		Tool:      sess.Tool,
	}

	// Try to extract slug from session file path or find plan
	if sess.Slug != "" {
		trailers.Slug = sess.Slug
	} else {
		// Try to find slug from recent plan files
		slug := findRecentPlanSlug(cwd)
		if slug != "" {
			trailers.Slug = slug
			sess.PlanFile = session.FindPlanFile(slug)
		}
	}

	// Add trailers to commit message
	if err := git.AddTrailersToFile(msgFile, trailers); err != nil {
		// Log error but don't block commit
		fmt.Fprintf(os.Stderr, "hunter: failed to add trailers: %v\n", err)
		return nil
	}

	return nil
}

// findRecentPlanSlug looks for a recently modified plan file
func findRecentPlanSlug(projectPath string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	plansDir := filepath.Join(homeDir, ".claude", "plans")
	entries, err := os.ReadDir(plansDir)
	if err != nil {
		return ""
	}

	// Find the most recently modified .md file
	var mostRecent string
	var mostRecentTime int64

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Unix() > mostRecentTime {
			mostRecentTime = info.ModTime().Unix()
			mostRecent = strings.TrimSuffix(entry.Name(), ".md")
		}
	}

	return mostRecent
}
