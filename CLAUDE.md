# Hunter - Claude Code Context

Hunter is a CLI tool and Chrome extension that traces code changes back to the AI sessions that created them, providing "git blame for AI."

## Build Commands

```bash
make build          # Build the hunter binary to build/hunter
make test           # Run tests with race detection
make lint           # Run go vet and golangci-lint
make fmt            # Format code with gofmt
make install        # Install to /usr/local/bin (requires sudo)
make extension      # Package Chrome extension
```

## Project Structure

```
hunter/
├── cmd/hunter/          # CLI commands (Cobra)
│   ├── main.go          # Entry point
│   ├── root.go          # Root command definition
│   ├── init.go          # Install git hook
│   ├── hook.go          # Hook subcommand (called by git)
│   ├── status.go        # Show session detection status
│   ├── blame.go         # Show line-level AI attribution
│   ├── session.go       # View session details
│   ├── restore.go       # Recreate session context
│   ├── enrich.go        # Add plan/summary to git notes
│   ├── push.go          # Push notes to remote
│   └── version.go       # Version command
├── internal/
│   ├── session/         # AI session detection
│   │   ├── detector.go  # Main detection logic
│   │   ├── process.go   # Find running Claude processes
│   │   ├── parser.go    # Parse session JSONL files
│   │   └── types.go     # Session type definitions
│   ├── hook/            # Git hook management
│   │   ├── installer.go # Install/uninstall hooks
│   │   └── script.go    # Hook script template
│   ├── storage/notes/   # Git notes read/write
│   └── git/             # Git operations
│       ├── repo.go      # Repository operations
│       └── trailers.go  # Git trailer handling
├── extension/           # Chrome extension
│   ├── manifest.json
│   ├── content/         # Content scripts for GitHub
│   ├── popup/           # Extension popup UI
│   └── icons/           # Extension icons (SVG)
├── go.mod
└── Makefile
```

## Key Packages

| Package | Purpose |
|---------|---------|
| `internal/session` | Detects active Claude sessions via running processes and session files in `~/.claude/projects/` |
| `internal/hook` | Installs and manages the `prepare-commit-msg` git hook |
| `internal/git` | Git operations: trailers, repo detection |
| `internal/storage/notes` | Read/write git notes at `refs/notes/ai` |

## Testing

```bash
make test                    # Run all tests
make test-coverage           # Run tests with coverage report
go test ./internal/session/  # Test specific package
```

Tests use the standard Go testing package. Key test files:
- `internal/session/detector_test.go` - Session detection tests
- `internal/git/trailers_test.go` - Git trailer parsing tests

## Code Style

- Standard Go idioms and conventions
- Cobra for CLI command structure
- Error handling: return errors up the stack, log at command level
- Package organization follows Go project layout conventions
- Internal packages for non-public code

## Session Detection

The core detection logic is in `internal/session/detector.go`:
1. First checks for running Claude processes with matching working directory
2. Falls back to checking for recently modified session files (< 15 min)
3. Session files are stored in `~/.claude/projects/[encoded-path]/`
4. Path encoding: `/Users/foo/project` → `-Users-foo-project`
