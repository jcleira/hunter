package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/arvos/hunter/internal/git"
	"github.com/arvos/hunter/internal/session"
	"github.com/arvos/hunter/internal/storage/notes"
	"github.com/spf13/cobra"
)

var enrichCmd = &cobra.Command{
	Use:   "enrich",
	Short: "Add plan/summary to git notes for commits with AI-Session trailers",
	Long: `Scan recent commits for AI-Session trailers and add detailed
information to git notes.

This command:
1. Finds commits with AI-Session trailers
2. Looks up the session plan from the slug
3. Optionally generates a summary
4. Stores the data in refs/notes/ai

The enriched data can then be viewed with 'hunter blame' or
the Chrome extension, and pushed to remotes with 'hunter push'.`,
	RunE: runEnrich,
}

var (
	enrichCount   int
	enrichDryRun  bool
	enrichVerbose bool
)

func init() {
	rootCmd.AddCommand(enrichCmd)
	enrichCmd.Flags().IntVarP(&enrichCount, "count", "n", 50, "Number of recent commits to scan")
	enrichCmd.Flags().BoolVar(&enrichDryRun, "dry-run", false, "Show what would be done without making changes")
	enrichCmd.Flags().BoolVarP(&enrichVerbose, "verbose", "v", false, "Show detailed progress")
}

func runEnrich(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	repo, err := git.Open(cwd)
	if err != nil {
		return fmt.Errorf("not a git repository: %w", err)
	}

	store := notes.NewStore(repo)

	// Get recent commits
	output, err := exec.Command("git", "-C", repo.Path, "log",
		fmt.Sprintf("-n%d", enrichCount),
		"--format=%H",
	).Output()
	if err != nil {
		return fmt.Errorf("failed to get commits: %w", err)
	}

	commits := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(commits) == 0 || (len(commits) == 1 && commits[0] == "") {
		fmt.Println("No commits found.")
		return nil
	}

	var enriched, skipped, noSession int

	for _, commit := range commits {
		if commit == "" {
			continue
		}

		// Check if already has note
		existing, _ := store.Get(commit)
		if existing != nil {
			skipped++
			if enrichVerbose {
				fmt.Printf("  %s: already enriched\n", shortID(commit))
			}
			continue
		}

		// Get commit message and check for AI trailers
		msg, err := repo.GetCommitMessage(commit)
		if err != nil {
			continue
		}

		trailers := git.ParseTrailersFromMessage(msg)
		if trailers.SessionID == "" {
			noSession++
			if enrichVerbose {
				fmt.Printf("  %s: no AI session\n", shortID(commit))
			}
			continue
		}

		// Found AI session - try to enrich
		if enrichVerbose {
			fmt.Printf("  %s: enriching (session %s)\n", shortID(commit), shortID(trailers.SessionID))
		}

		// Find the session
		sess, err := session.FindSessionByID(cwd, trailers.SessionID)
		if err != nil {
			// Session file might not exist anymore, create minimal note
			sess = &session.Session{
				ID:   trailers.SessionID,
				Slug: trailers.Slug,
				Tool: trailers.Tool,
			}
			if sess.Tool == "" {
				sess.Tool = "claude"
			}
		}

		// If we have a slug, try to find the plan
		if trailers.Slug != "" && sess.PlanFile == "" {
			sess.PlanFile = session.FindPlanFile(trailers.Slug)
			sess.Slug = trailers.Slug
		}

		// Get files changed in this commit
		filesOutput, _ := exec.Command("git", "-C", repo.Path, "diff-tree",
			"--no-commit-id", "--name-only", "-r", commit,
		).Output()
		filesChanged := strings.Split(strings.TrimSpace(string(filesOutput)), "\n")
		if len(filesChanged) == 1 && filesChanged[0] == "" {
			filesChanged = nil
		}

		// Build note data
		noteData := notes.BuildNoteData(sess, filesChanged, "")

		if enrichDryRun {
			fmt.Printf("Would enrich %s with session %s\n", shortID(commit), shortID(trailers.SessionID))
			if sess.PlanFile != "" {
				fmt.Printf("  Plan: %s\n", sess.PlanFile)
			}
			enriched++
			continue
		}

		// Store the note
		if err := store.Set(commit, noteData); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to enrich %s: %v\n", shortID(commit), err)
			continue
		}

		enriched++
	}

	fmt.Println()
	fmt.Printf("Scanned %d commits:\n", len(commits))
	fmt.Printf("  Enriched:   %d\n", enriched)
	fmt.Printf("  Skipped:    %d (already enriched)\n", skipped)
	fmt.Printf("  No session: %d\n", noSession)

	if enriched > 0 && !enrichDryRun {
		fmt.Println()
		fmt.Println("Use 'hunter push <remote>' to sync notes with remote.")
	}

	return nil
}
