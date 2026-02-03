package main

import (
	"fmt"
	"os"

	"github.com/arvos/hunter/internal/git"
	"github.com/arvos/hunter/internal/hook"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Install git hook for current repo",
	Long: `Install the hunter git hook in the current repository.

This hook runs at commit time and:
1. Detects if an AI session (like Claude Code) is active
2. Adds AI-Session and AI-Slug trailers to the commit message
3. Links your commits to the AI sessions that created them

The hook is non-blocking - if hunter fails or no session is detected,
the commit proceeds normally.`,
	RunE: runInit,
}

var (
	initUninstall bool
	initForce     bool
)

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVarP(&initUninstall, "uninstall", "u", false, "Uninstall the hook")
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Force installation even if another hook exists")
}

func runInit(cmd *cobra.Command, args []string) error {
	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Open the git repo
	repo, err := git.Open(cwd)
	if err != nil {
		return fmt.Errorf("not a git repository: %w", err)
	}

	installer := hook.NewInstaller(repo)

	if initUninstall {
		return runUninstall(installer, repo)
	}

	return runInstall(installer, repo)
}

func runInstall(installer *hook.Installer, repo *git.Repo) error {
	hookName := "prepare-commit-msg"

	// Check current status
	status := installer.Status(hookName)
	if status == "installed" {
		fmt.Println("Hunter hook is already installed.")
		return nil
	}

	// Install the hook
	if err := installer.Install(hookName); err != nil {
		if err == hook.ErrHookExists && !initForce {
			fmt.Println("A git hook already exists at .git/hooks/prepare-commit-msg")
			fmt.Println("Use --force to chain hunter with the existing hook")
			return err
		}
		return fmt.Errorf("failed to install hook: %w", err)
	}

	fmt.Println("Hunter hook installed successfully!")
	fmt.Printf("  Hook location: %s/hooks/%s\n", repo.GitDir, hookName)
	fmt.Println()
	fmt.Println("Now when you commit, hunter will automatically:")
	fmt.Println("  - Detect active AI sessions (Claude Code)")
	fmt.Println("  - Add AI-Session trailers to commit messages")
	fmt.Println()
	fmt.Println("Run 'hunter status' to check session detection.")

	return nil
}

func runUninstall(installer *hook.Installer, repo *git.Repo) error {
	hookName := "prepare-commit-msg"

	if err := installer.Uninstall(hookName); err != nil {
		return fmt.Errorf("failed to uninstall hook: %w", err)
	}

	fmt.Println("Hunter hook uninstalled.")
	return nil
}
