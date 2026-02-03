package hook

// PrepareCommitMsgScript is the git hook script for prepare-commit-msg
const PrepareCommitMsgScript = `#!/bin/sh
# Hunter: Git blame for AI - https://github.com/arvos/hunter
# This hook detects active AI sessions and adds trailers to commit messages

# Only process regular commits (not merge, squash, etc.)
case "$2" in
    merge|squash)
        exit 0
        ;;
    message|template|"")
        # These are the cases we want to handle
        ;;
    *)
        exit 0
        ;;
esac

# Run hunter hook command
# Silence errors so we don't block commits if hunter has issues
hunter hook prepare-commit-msg "$1" 2>/dev/null || true
`

// GetHookScript returns the hook script for the given hook name
func GetHookScript(hookName string) string {
	switch hookName {
	case "prepare-commit-msg":
		return PrepareCommitMsgScript
	default:
		return ""
	}
}
