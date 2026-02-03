package git

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	// ErrNotGitRepo indicates the directory is not a git repository
	ErrNotGitRepo = errors.New("not a git repository")
)

// Repo represents a git repository
type Repo struct {
	// Path is the root directory of the repository
	Path string
	// GitDir is the path to the .git directory
	GitDir string
}

// Open opens a git repository at the given path
func Open(path string) (*Repo, error) {
	// Find the repository root
	cmd := exec.Command("git", "-C", path, "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return nil, ErrNotGitRepo
	}
	repoPath := strings.TrimSpace(string(output))

	// Find the git directory
	cmd = exec.Command("git", "-C", repoPath, "rev-parse", "--git-dir")
	output, err = cmd.Output()
	if err != nil {
		return nil, ErrNotGitRepo
	}
	gitDir := strings.TrimSpace(string(output))

	// Make git dir absolute if relative
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(repoPath, gitDir)
	}

	return &Repo{
		Path:   repoPath,
		GitDir: gitDir,
	}, nil
}

// HooksDir returns the path to the hooks directory
func (r *Repo) HooksDir() string {
	return filepath.Join(r.GitDir, "hooks")
}

// CurrentBranch returns the current branch name
func (r *Repo) CurrentBranch() (string, error) {
	cmd := exec.Command("git", "-C", r.Path, "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// Head returns the current HEAD commit hash
func (r *Repo) Head() (string, error) {
	cmd := exec.Command("git", "-C", r.Path, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// GetCommitMessage returns the commit message for a given commit
func (r *Repo) GetCommitMessage(commit string) (string, error) {
	cmd := exec.Command("git", "-C", r.Path, "log", "-1", "--format=%B", commit)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// GetTrailer returns a trailer value from a commit
func (r *Repo) GetTrailer(commit, key string) (string, error) {
	cmd := exec.Command("git", "-C", r.Path, "log", "-1", "--format=%(trailers:key="+key+",valueonly)", commit)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// ListCommitsWithTrailer lists commits that have a specific trailer
func (r *Repo) ListCommitsWithTrailer(key string, count int) ([]string, error) {
	args := []string{"-C", r.Path, "log", "--format=%H %(trailers:key=" + key + ",valueonly)"}
	if count > 0 {
		args = append(args, "-n", string(rune(count+'0')))
	}

	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var commits []string
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 && parts[1] != "" {
			commits = append(commits, parts[0])
		}
	}

	return commits, nil
}

// Blame returns blame information for a file
func (r *Repo) Blame(file string) ([]BlameLine, error) {
	cmd := exec.Command("git", "-C", r.Path, "blame", "--porcelain", file)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return parseBlame(string(output)), nil
}

// BlameLine represents a single line of blame output
type BlameLine struct {
	Commit     string
	LineNumber int
	Content    string
}

// parseBlame parses porcelain blame output
func parseBlame(output string) []BlameLine {
	var lines []BlameLine
	var currentCommit string
	lineNum := 0

	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}

		// Lines starting with a hash are commit headers
		if len(line) >= 40 && !strings.HasPrefix(line, "\t") {
			parts := strings.Fields(line)
			if len(parts) >= 1 && len(parts[0]) == 40 {
				currentCommit = parts[0]
				lineNum++
			}
		}

		// Lines starting with tab are content
		if strings.HasPrefix(line, "\t") {
			lines = append(lines, BlameLine{
				Commit:     currentCommit,
				LineNumber: lineNum,
				Content:    strings.TrimPrefix(line, "\t"),
			})
		}
	}

	return lines
}

// GetNote returns the content of a git note for a commit
func (r *Repo) GetNote(commit, ref string) (string, error) {
	cmd := exec.Command("git", "-C", r.Path, "notes", "--ref="+ref, "show", commit)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		// Check if note doesn't exist
		if strings.Contains(stderr.String(), "No note found") {
			return "", nil
		}
		return "", err
	}
	return string(output), nil
}

// SetNote sets a git note for a commit
func (r *Repo) SetNote(commit, ref, content string) error {
	cmd := exec.Command("git", "-C", r.Path, "notes", "--ref="+ref, "add", "-f", "-m", content, commit)
	return cmd.Run()
}

// PushNotes pushes notes to a remote
func (r *Repo) PushNotes(remote, ref string) error {
	cmd := exec.Command("git", "-C", r.Path, "push", remote, "refs/notes/"+ref)
	return cmd.Run()
}

// FetchNotes fetches notes from a remote
func (r *Repo) FetchNotes(remote, ref string) error {
	cmd := exec.Command("git", "-C", r.Path, "fetch", remote, "refs/notes/"+ref+":refs/notes/"+ref)
	return cmd.Run()
}

// GetCurrentDir returns the current working directory
func GetCurrentDir() (string, error) {
	return os.Getwd()
}
