package main

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "hunter",
	Short: "Git blame for AI - trace code changes back to AI sessions",
	Long: `Hunter is a CLI tool that traces code changes back to the AI sessions that created them.

It works by installing a git hook that detects active AI sessions at commit time,
tagging commits with session information, and providing commands to query AI
attribution for any line of code.

Commands:
  init      Install git hook for current repo
  status    Show current session detection status
  blame     Show AI attribution per line
  session   View session details (plan, summary, files)
  restore   Recreate session context locally
  enrich    Add plan/summary to git notes for commits with AI-Session trailers
  push      Push refs/notes/ai to remote`,
}

func init() {
	// Commands are added in their respective files
}

// shortID returns the first 8 characters of an ID, or the full ID if shorter
func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}
