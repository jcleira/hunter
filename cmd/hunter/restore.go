package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/arvos/hunter/internal/session"
	"github.com/spf13/cobra"
)

var restoreCmd = &cobra.Command{
	Use:   "restore <session-id>",
	Short: "Recreate session context locally",
	Long: `Restore the context for a previous AI session.

This helps you understand or continue work from a previous session by:
- Showing the session's plan file
- Locating the session log
- Optionally checking out the git state from that session

Session IDs can be partial (e.g., "abc123" instead of the full UUID).`,
	Args: cobra.ExactArgs(1),
	RunE: runRestore,
}

var (
	restoreCheckout bool
	restoreOpen     bool
	restoreExport   bool
)

func init() {
	rootCmd.AddCommand(restoreCmd)
	restoreCmd.Flags().BoolVar(&restoreCheckout, "checkout", false, "Checkout git state at session start")
	restoreCmd.Flags().BoolVar(&restoreOpen, "open", false, "Open session in editor (if supported)")
	restoreCmd.Flags().BoolVar(&restoreExport, "export", false, "Export session to shareable format")
}

func runRestore(cmd *cobra.Command, args []string) error {
	sessionID := args[0]

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find the session
	sess, err := session.FindSessionByID(cwd, sessionID)
	if err != nil {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	fmt.Println("Restoring session context...")
	fmt.Println()

	// Show session info
	fmt.Printf("├── Session ID: %s\n", sess.ID)
	fmt.Printf("├── Tool: %s\n", sess.Tool)

	if sess.PlanFile != "" {
		fmt.Printf("├── Plan file: %s\n", sess.PlanFile)
	} else {
		fmt.Printf("├── Plan file: (none found)\n")
	}

	fmt.Printf("├── Session log: %s\n", sess.SessionFile)

	// Check if session file exists and show size
	if info, err := os.Stat(sess.SessionFile); err == nil {
		fmt.Printf("│   └── Size: %d KB\n", info.Size()/1024)
	}

	fmt.Println()

	if restoreCheckout {
		fmt.Println("--checkout: Git checkout not yet implemented")
		fmt.Println("            Would checkout to the first commit in this session")
	}

	if restoreOpen {
		return openInEditor(sess)
	}

	if restoreExport {
		return exportSession(sess)
	}

	// Show available actions
	fmt.Println("Available actions:")
	fmt.Println()

	if sess.PlanFile != "" {
		fmt.Printf("  View plan:    cat %s\n", sess.PlanFile)
	}
	fmt.Printf("  View session: cat %s | head -100\n", sess.SessionFile)
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --open      Open session log in editor")
	fmt.Println("  --export    Export session to shareable JSON")

	return nil
}

func openInEditor(sess *session.Session) error {
	// Try common editors
	editors := []string{
		os.Getenv("EDITOR"),
		"code",
		"vim",
		"nano",
	}

	for _, editor := range editors {
		if editor == "" {
			continue
		}

		// Check if editor exists
		if _, err := exec.LookPath(editor); err != nil {
			continue
		}

		// Open the session file
		cmd := exec.Command(editor, sess.SessionFile)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		return cmd.Run()
	}

	return fmt.Errorf("no suitable editor found (set $EDITOR)")
}

func exportSession(sess *session.Session) error {
	// Build export structure
	export := struct {
		SessionID   string `json:"session_id"`
		Tool        string `json:"tool"`
		Model       string `json:"model,omitempty"`
		PlanFile    string `json:"plan_file,omitempty"`
		PlanContent string `json:"plan_content,omitempty"`
		SessionFile string `json:"session_file"`
		ProjectPath string `json:"project_path"`
	}{
		SessionID:   sess.ID,
		Tool:        sess.Tool,
		Model:       sess.Model,
		PlanFile:    sess.PlanFile,
		SessionFile: sess.SessionFile,
		ProjectPath: sess.ProjectPath,
	}

	// Read plan content if available
	if sess.PlanFile != "" {
		if content, err := session.ReadPlanContent(sess.PlanFile); err == nil {
			export.PlanContent = content
		}
	}

	// Output as JSON
	fmt.Println("{")
	fmt.Printf("  \"session_id\": %q,\n", export.SessionID)
	fmt.Printf("  \"tool\": %q,\n", export.Tool)
	if export.Model != "" {
		fmt.Printf("  \"model\": %q,\n", export.Model)
	}
	if export.PlanFile != "" {
		fmt.Printf("  \"plan_file\": %q,\n", export.PlanFile)
	}
	if export.PlanContent != "" {
		// Escape the plan content for JSON
		fmt.Printf("  \"plan_content\": %q,\n", export.PlanContent)
	}
	fmt.Printf("  \"session_file\": %q,\n", export.SessionFile)
	fmt.Printf("  \"project_path\": %q\n", export.ProjectPath)
	fmt.Println("}")

	return nil
}
