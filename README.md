# Hunter

**Git blame for AI** — Trace code changes back to the AI sessions that created them.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## The Problem

When reviewing code, `git blame` shows who made each change and when. But with AI-assisted coding, the real question is: **what prompt or conversation led to this code?**

Hunter answers that by linking commits to AI sessions.

## How It Works

```
┌─────────────────┐     ┌─────────────┐     ┌──────────────┐
│  AI Tool Data   │     │  Git Repo   │     │   Storage    │
│  ~/.claude/     │────▶│  commits    │────▶│  git notes   │
└─────────────────┘     └─────────────┘     └──────────────┘
         │                     │                    │
         └─────────────────────┼────────────────────┘
                               ▼
                    ┌─────────────────────┐
                    │    Hunter CLI       │
                    │  init │ blame │ ... │
                    └─────────────────────┘
                               │
                               ▼
                    ┌─────────────────────┐
                    │  Chrome Extension   │
                    │  GitHub PR/Blame    │
                    └─────────────────────┘
```

1. **Hook**: Git hook detects active AI session at commit time
2. **Tag**: Adds `AI-Session` trailer to commit message automatically
3. **Enrich**: Later, add plan + summary to git notes
4. **Visualize**: Chrome extension shows AI attribution on GitHub

## Installation

### Build from Source

```bash
git clone https://github.com/arvos/hunter.git
cd hunter
make build
make install  # Installs to /usr/local/bin
```

### Homebrew (coming soon)

```bash
brew install hunter
```

## Quick Start

```bash
# 1. Initialize hunter in your repo
cd your-project
hunter init

# 2. Work with Claude Code as normal
# When you commit, hunter automatically detects the active session
# and adds AI-Session trailers to your commit message

# 3. Check if a session is detected
hunter status

# 4. View AI attribution for a file
hunter blame src/main.go
```

## CLI Commands

### `hunter init`

Install the git hook for the current repository.

```bash
hunter init
# Output: Installed git hook at .git/hooks/prepare-commit-msg
```

### `hunter status`

Show current session detection status.

```bash
hunter status
# Output:
# Active AI Session Detected
#   Session ID: 72a5de1c-ae47-4df9-afb6-ad71c63d7049
#   Tool: claude
#   Slug: tender-snacking-unicorn
#   Last Activity: 2 minutes ago
```

### `hunter blame <file>`

Show AI attribution per line, similar to `git blame`.

```bash
hunter blame src/auth/jwt.go
# Output shows which lines were written with AI assistance
```

### `hunter session <id>`

View details of a specific session.

```bash
hunter session 72a5de1c
# Shows: plan, summary, files changed, session path
```

### `hunter restore <id>`

Recreate session context locally.

```bash
hunter restore 72a5de1c
# Restores plan file and shows session information
```

### `hunter enrich`

Add plan and summary to git notes for commits with AI-Session trailers.

```bash
hunter enrich
# Reads commits with AI-Session trailers and enriches with notes
```

### `hunter push <remote>`

Push AI notes to a remote repository.

```bash
hunter push origin
# Pushes refs/notes/ai to the remote
```

## Chrome Extension

The Chrome extension highlights AI-assisted code on GitHub.

### Installation

1. Build the extension:
   ```bash
   make extension
   ```

2. Open Chrome and go to `chrome://extensions/`

3. Enable **Developer mode** (toggle in top right)

4. Click **Load unpacked** and select the `extension/` directory

### Features

- Orange dot indicator on lines written with AI assistance
- Hover to see session plan and summary
- "Open locally" link to restore session context
- Works on GitHub PR diffs and blame views

## What Gets Stored

Hunter stores curated context in git notes:

```json
{
  "session_id": "72a5de1c-ae47-4df9-afb6-ad71c63d7049",
  "tool": "claude",
  "model": "claude-opus-4-5-20251101",
  "plan": "## Plan\n1. Add JWT auth middleware...",
  "summary": "Implemented JWT authentication with refresh tokens.",
  "files_changed": ["src/auth/jwt.go", "src/middleware/auth.go"]
}
```

Commit messages include trailers:

```
Add JWT authentication

AI-Session: 72a5de1c-ae47-4df9-afb6-ad71c63d7049
AI-Slug: tender-snacking-unicorn
```

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests (`make test`)
5. Commit your changes
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

## License

MIT License - see [LICENSE](LICENSE) for details.
