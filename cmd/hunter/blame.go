package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/arvos/hunter/internal/git"
	"github.com/spf13/cobra"
)

var blameCmd = &cobra.Command{
	Use:   "blame <file>",
	Short: "Show AI attribution per line",
	Long: `Show which lines in a file were created in AI-assisted commits.

Like git blame, but highlights lines that came from AI sessions
and shows the session information.`,
	Args: cobra.ExactArgs(1),
	RunE: runBlame,
}

var (
	blameShowAll  bool
	blameJSON     bool
	blameShowPlan bool
)

func init() {
	rootCmd.AddCommand(blameCmd)
	blameCmd.Flags().BoolVarP(&blameShowAll, "all", "a", false, "Show all lines, not just AI-attributed ones")
	blameCmd.Flags().BoolVarP(&blameJSON, "json", "j", false, "Output as JSON")
	blameCmd.Flags().BoolVarP(&blameShowPlan, "plan", "p", false, "Show plan content for AI commits")
}

type blameLine struct {
	LineNumber int    `json:"line"`
	Content    string `json:"content"`
	Commit     string `json:"commit"`
	IsAI       bool   `json:"is_ai"`
	SessionID  string `json:"session_id,omitempty"`
	Slug       string `json:"slug,omitempty"`
}

func runBlame(cmd *cobra.Command, args []string) error {
	file := args[0]

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	repo, err := git.Open(cwd)
	if err != nil {
		return fmt.Errorf("not a git repository: %w", err)
	}

	// Get git blame for the file
	blameLines, err := repo.Blame(file)
	if err != nil {
		return fmt.Errorf("failed to blame file: %w", err)
	}

	// Cache of commit -> AI trailers
	commitTrailers := make(map[string]*git.Trailers)

	var output []blameLine
	var aiLines, totalLines int

	for _, line := range blameLines {
		totalLines++

		// Get trailers for this commit (cached)
		trailers, ok := commitTrailers[line.Commit]
		if !ok {
			msg, err := repo.GetCommitMessage(line.Commit)
			if err != nil {
				trailers = &git.Trailers{}
			} else {
				trailers = git.ParseTrailersFromMessage(msg)
			}
			commitTrailers[line.Commit] = trailers
		}

		isAI := trailers.SessionID != ""
		if isAI {
			aiLines++
		}

		bl := blameLine{
			LineNumber: line.LineNumber,
			Content:    line.Content,
			Commit:     line.Commit[:8],
			IsAI:       isAI,
			SessionID:  trailers.SessionID,
			Slug:       trailers.Slug,
		}

		if blameShowAll || isAI {
			output = append(output, bl)
		}
	}

	if blameJSON {
		return outputBlameJSON(output, aiLines, totalLines)
	}

	return outputBlameText(output, aiLines, totalLines, file)
}

func outputBlameJSON(lines []blameLine, aiLines, totalLines int) error {
	result := struct {
		Lines      []blameLine `json:"lines"`
		AILines    int         `json:"ai_lines"`
		TotalLines int         `json:"total_lines"`
		AIPercent  float64     `json:"ai_percent"`
	}{
		Lines:      lines,
		AILines:    aiLines,
		TotalLines: totalLines,
		AIPercent:  float64(aiLines) / float64(totalLines) * 100,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func outputBlameText(lines []blameLine, aiLines, totalLines int, file string) error {
	// Calculate AI percentage
	aiPercent := float64(aiLines) / float64(totalLines) * 100

	// Header
	fmt.Printf("%s  [AI: %.0f%%]\n", file, aiPercent)
	fmt.Println(strings.Repeat("-", 60))

	if len(lines) == 0 {
		if !blameShowAll {
			fmt.Println("No AI-attributed lines found.")
			fmt.Println("Use --all to show all lines.")
		}
		return nil
	}

	// Output lines
	for _, line := range lines {
		marker := " "
		if line.IsAI {
			marker = "●"
		}

		// Truncate content for display
		content := line.Content
		if len(content) > 50 {
			content = content[:47] + "..."
		}

		if blameShowAll {
			fmt.Printf("%4d %s│ %s │ %s\n", line.LineNumber, marker, line.Commit, content)
		} else {
			// AI-only mode - show more detail
			fmt.Printf("%4d %s│ %s │ %s\n", line.LineNumber, marker, line.Commit, content)
			if line.Slug != "" {
				fmt.Printf("        └─ session: %s\n", line.Slug)
			}
		}
	}

	fmt.Println()
	fmt.Printf("● = AI-assisted (%d of %d lines, %.0f%%)\n", aiLines, totalLines, aiPercent)

	return nil
}
