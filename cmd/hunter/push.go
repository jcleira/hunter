package main

import (
	"fmt"
	"os"

	"github.com/arvos/hunter/internal/git"
	"github.com/arvos/hunter/internal/storage/notes"
	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push <remote>",
	Short: "Push refs/notes/ai to remote",
	Long: `Push AI session notes to a remote repository.

This allows collaborators to see AI attribution when they clone or
pull the repository. The Chrome extension will also be able to
display session information on GitHub.

Example:
  hunter push origin`,
	Args: cobra.ExactArgs(1),
	RunE: runPush,
}

var (
	pushFetch bool
)

func init() {
	rootCmd.AddCommand(pushCmd)
	pushCmd.Flags().BoolVarP(&pushFetch, "fetch", "f", false, "Fetch notes from remote before pushing")
}

func runPush(cmd *cobra.Command, args []string) error {
	remote := args[0]

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	repo, err := git.Open(cwd)
	if err != nil {
		return fmt.Errorf("not a git repository: %w", err)
	}

	store := notes.NewStore(repo)

	// Optionally fetch first
	if pushFetch {
		fmt.Printf("Fetching notes from %s...\n", remote)
		if err := store.Fetch(remote); err != nil {
			// Fetch failure is not fatal - notes might not exist yet
			fmt.Printf("  Note: fetch failed (notes may not exist on remote yet)\n")
		}
	}

	// Push notes
	fmt.Printf("Pushing refs/notes/ai to %s...\n", remote)
	if err := store.Push(remote); err != nil {
		return fmt.Errorf("failed to push notes: %w", err)
	}

	fmt.Println("Done! AI session notes pushed to remote.")
	fmt.Println()
	fmt.Println("Collaborators can fetch notes with:")
	fmt.Printf("  git fetch %s refs/notes/ai:refs/notes/ai\n", remote)

	return nil
}

// Add fetch command as well
var fetchCmd = &cobra.Command{
	Use:   "fetch <remote>",
	Short: "Fetch refs/notes/ai from remote",
	Long: `Fetch AI session notes from a remote repository.

This downloads AI attribution data from the remote so you can
view session information with 'hunter blame' or the Chrome extension.

Example:
  hunter fetch origin`,
	Args: cobra.ExactArgs(1),
	RunE: runFetch,
}

func init() {
	rootCmd.AddCommand(fetchCmd)
}

func runFetch(cmd *cobra.Command, args []string) error {
	remote := args[0]

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	repo, err := git.Open(cwd)
	if err != nil {
		return fmt.Errorf("not a git repository: %w", err)
	}

	store := notes.NewStore(repo)

	fmt.Printf("Fetching refs/notes/ai from %s...\n", remote)
	if err := store.Fetch(remote); err != nil {
		return fmt.Errorf("failed to fetch notes: %w", err)
	}

	fmt.Println("Done! AI session notes fetched from remote.")
	fmt.Println()
	fmt.Println("Use 'hunter blame <file>' to see AI attribution.")

	return nil
}
