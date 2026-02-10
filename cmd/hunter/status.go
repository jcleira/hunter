package main

import (
	"fmt"
	"os"
	"time"

	"github.com/arvos/hunter/internal/git"
	"github.com/arvos/hunter/internal/hook"
	"github.com/arvos/hunter/internal/session"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current session detection status",
	Long: `Show whether an AI session is currently active for this project.

This helps verify that hunter is correctly detecting your AI coding sessions
before making commits.`,
	RunE: runStatus,
}

var statusDebug bool

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().BoolVarP(&statusDebug, "debug", "d", false, "Show debug information")
}

func runStatus(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Check git repo
	repo, err := git.Open(cwd)
	if err != nil {
		fmt.Println("Not a git repository")
		return nil
	}

	// Check hook installation
	installer := hook.NewInstaller(repo)
	hookStatus := installer.Status("prepare-commit-msg")

	fmt.Println("Hunter Status")
	fmt.Println("=============")
	fmt.Println()
	fmt.Printf("Repository: %s\n", repo.Path)
	fmt.Printf("Hook:       %s\n", hookStatus)
	fmt.Println()

	// Debug: show all sessions if debug flag is set
	if statusDebug {
		fmt.Println("Debug: Looking for sessions...")
		sessions, dbgErr := session.FindAllSessions(cwd)
		if dbgErr != nil {
			fmt.Printf("Debug: Error finding sessions: %v\n", dbgErr)
		} else {
			fmt.Printf("Debug: Found %d sessions\n", len(sessions))
			for i, s := range sessions {
				fmt.Printf("Debug: Session %d: ID=%s, LastActivity=%v, Age=%v\n",
					i, shortID(s.ID), s.LastActivity, time.Since(s.LastActivity))
			}
		}
		fmt.Println()
	}

	// Check for active session
	sess, err := session.DetectActiveSession(cwd)
	if err != nil {
		fmt.Println("AI Session: none detected")
		fmt.Println()
		fmt.Println("No active AI session found. This could mean:")
		fmt.Println("  - No AI tool is currently running in this project")
		fmt.Println("  - The session was last active more than 15 minutes ago")
		fmt.Println("  - This is the first time using an AI tool in this project")
		return nil
	}

	fmt.Println("AI Session: ACTIVE")
	fmt.Printf("  ID:       %s\n", sess.ID)
	if sess.Slug != "" {
		fmt.Printf("  Slug:     %s\n", sess.Slug)
	}
	fmt.Printf("  Tool:     %s\n", sess.Tool)
	if sess.Model != "" {
		fmt.Printf("  Model:    %s\n", sess.Model)
	}
	fmt.Printf("  Active:   %s ago\n", formatDuration(time.Since(sess.LastActivity)))
	if sess.IsRunning {
		fmt.Println("  Process:  running")
	}
	fmt.Printf("  File:     %s\n", sess.SessionFile)
	if sess.PlanFile != "" {
		fmt.Printf("  Plan:     %s\n", sess.PlanFile)
	}

	fmt.Println()
	fmt.Println("Commits made now will be tagged with this session.")

	return nil
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d seconds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	}
	return fmt.Sprintf("%d hours", int(d.Hours()))
}
