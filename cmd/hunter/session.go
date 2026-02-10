package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/arvos/hunter/internal/session"
	"github.com/spf13/cobra"
)

var sessionCmd = &cobra.Command{
	Use:   "session [id]",
	Short: "View session details",
	Long: `View details about an AI session including the plan, summary, and files changed.

If no session ID is provided, shows the current active session.
Session IDs can be partial (e.g., "abc123" instead of the full UUID).`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSession,
}

var (
	sessionList bool
	sessionJSON bool
)

func init() {
	rootCmd.AddCommand(sessionCmd)
	sessionCmd.Flags().BoolVarP(&sessionList, "list", "l", false, "List all sessions for this project")
	sessionCmd.Flags().BoolVarP(&sessionJSON, "json", "j", false, "Output as JSON")
}

func runSession(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if sessionList {
		return listSessions(cwd)
	}

	// Get specific session or current
	var sess *session.Session
	if len(args) > 0 {
		sess, err = session.FindSessionByID(cwd, args[0])
		if err != nil {
			return fmt.Errorf("session not found: %s", args[0])
		}
	} else {
		sess, err = session.DetectActiveSession(cwd)
		if err != nil {
			return fmt.Errorf("no active session found")
		}
	}

	if sessionJSON {
		return outputSessionJSON(sess)
	}

	return outputSessionText(sess)
}

func listSessions(projectPath string) error {
	sessions, err := session.FindAllSessions(projectPath)
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions found for this project.")
		return nil
	}

	if sessionJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(sessions)
	}

	fmt.Println("Sessions for this project:")
	fmt.Println(strings.Repeat("-", 60))

	for _, sess := range sessions {
		status := "inactive"
		if sess.IsRunning {
			status = "running"
		} else if time.Since(sess.LastActivity) < session.SessionTimeout {
			status = "active"
		}

		fmt.Printf("  %s  %-10s  %s ago\n",
			shortID(sess.ID),
			status,
			formatDuration(time.Since(sess.LastActivity)))

		if sess.Slug != "" {
			fmt.Printf("           slug: %s\n", sess.Slug)
		}
	}

	fmt.Println()
	fmt.Println("Use 'hunter session <id>' to view details.")

	return nil
}

func outputSessionJSON(sess *session.Session) error {
	// Build output structure
	output := struct {
		*session.Session
		PlanContent string `json:"plan_content,omitempty"`
	}{
		Session: sess,
	}

	if sess.PlanFile != "" {
		if content, err := session.ReadPlanContent(sess.PlanFile); err == nil {
			output.PlanContent = content
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

func outputSessionText(sess *session.Session) error {
	fmt.Println("Session Details")
	fmt.Println("===============")
	fmt.Println()

	fmt.Printf("ID:           %s\n", sess.ID)
	if sess.Slug != "" {
		fmt.Printf("Slug:         %s\n", sess.Slug)
	}
	fmt.Printf("Tool:         %s\n", sess.Tool)
	if sess.Model != "" {
		fmt.Printf("Model:        %s\n", sess.Model)
	}
	fmt.Printf("Project:      %s\n", sess.ProjectPath)
	fmt.Printf("Session File: %s\n", sess.SessionFile)
	if !sess.StartTime.IsZero() {
		fmt.Printf("Started:      %s\n", sess.StartTime.Format(time.RFC3339))
	}
	fmt.Printf("Last Active:  %s (%s ago)\n",
		sess.LastActivity.Format(time.RFC3339),
		formatDuration(time.Since(sess.LastActivity)))

	if sess.IsRunning {
		fmt.Printf("Status:       running\n")
	}

	// Show plan content if available
	if sess.PlanFile != "" {
		fmt.Println()
		fmt.Printf("Plan File: %s\n", sess.PlanFile)
		fmt.Println(strings.Repeat("-", 40))

		content, err := session.ReadPlanContent(sess.PlanFile)
		if err != nil {
			fmt.Printf("(could not read plan: %v)\n", err)
		} else {
			// Limit plan output
			lines := strings.Split(content, "\n")
			if len(lines) > 30 {
				for _, line := range lines[:30] {
					fmt.Println(line)
				}
				fmt.Printf("\n... (%d more lines)\n", len(lines)-30)
			} else {
				fmt.Println(content)
			}
		}
	}

	fmt.Println()
	fmt.Println("Commands:")
	fmt.Printf("  hunter restore %s    # Recreate session context\n", shortID(sess.ID))

	return nil
}
