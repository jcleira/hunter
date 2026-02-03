package git

import (
	"bufio"
	"os"
	"strings"
)

const (
	// TrailerSessionID is the trailer key for session ID
	TrailerSessionID = "AI-Session"
	// TrailerSlug is the trailer key for session slug
	TrailerSlug = "AI-Slug"
	// TrailerTool is the trailer key for the AI tool name
	TrailerTool = "AI-Tool"
)

// Trailers represents the AI-related trailers for a commit
type Trailers struct {
	SessionID string
	Slug      string
	Tool      string
}

// AddTrailersToFile adds AI trailers to a commit message file
func AddTrailersToFile(msgFile string, trailers *Trailers) error {
	// Read the existing message
	content, err := os.ReadFile(msgFile)
	if err != nil {
		return err
	}

	msg := string(content)

	// Check if trailers already exist
	if strings.Contains(msg, TrailerSessionID+":") {
		return nil // Already has AI trailers
	}

	// Build the new message with trailers
	newMsg := addTrailersToMessage(msg, trailers)

	// Write back
	return os.WriteFile(msgFile, []byte(newMsg), 0644)
}

// addTrailersToMessage adds trailers to a commit message string
func addTrailersToMessage(msg string, trailers *Trailers) string {
	// Trim trailing whitespace and get the message body
	msg = strings.TrimRight(msg, "\n\t ")

	// Check if message already has trailers (lines at the end with "Key: Value" format)
	lines := strings.Split(msg, "\n")
	hasExistingTrailers := false
	trailerStartIdx := len(lines)

	// Find where trailers start (if any)
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			trailerStartIdx = i + 1
			break
		}
		if isTrailerLine(line) {
			hasExistingTrailers = true
		} else {
			// Non-trailer, non-empty line found - no existing trailers
			trailerStartIdx = len(lines)
			break
		}
	}

	// Build the trailer block
	var trailerLines []string
	if trailers.SessionID != "" {
		trailerLines = append(trailerLines, TrailerSessionID+": "+trailers.SessionID)
	}
	if trailers.Slug != "" {
		trailerLines = append(trailerLines, TrailerSlug+": "+trailers.Slug)
	}
	if trailers.Tool != "" {
		trailerLines = append(trailerLines, TrailerTool+": "+trailers.Tool)
	}

	if len(trailerLines) == 0 {
		return msg + "\n"
	}

	// Build the final message
	var result strings.Builder

	// Write the message body (everything before trailers)
	bodyLines := lines[:trailerStartIdx]
	for i, line := range bodyLines {
		result.WriteString(line)
		if i < len(bodyLines)-1 {
			result.WriteString("\n")
		}
	}

	// Add blank line before trailers if there isn't one and we have body content
	if len(bodyLines) > 0 && strings.TrimSpace(bodyLines[len(bodyLines)-1]) != "" {
		result.WriteString("\n\n")
	} else if len(bodyLines) > 0 {
		result.WriteString("\n")
	}

	// Add any existing trailers
	if hasExistingTrailers {
		for i := trailerStartIdx; i < len(lines); i++ {
			result.WriteString(lines[i])
			result.WriteString("\n")
		}
	}

	// Add our new trailers
	for _, trailer := range trailerLines {
		result.WriteString(trailer)
		result.WriteString("\n")
	}

	return result.String()
}

// isTrailerLine checks if a line looks like a git trailer
func isTrailerLine(line string) bool {
	// Trailers are "Key: Value" format
	// Key must start with a letter and contain only alphanumeric chars and hyphens
	colonIdx := strings.Index(line, ": ")
	if colonIdx <= 0 {
		return false
	}

	key := line[:colonIdx]
	if len(key) == 0 {
		return false
	}

	// First char must be a letter
	if !isLetter(key[0]) {
		return false
	}

	// Rest must be alphanumeric or hyphen
	for i := 1; i < len(key); i++ {
		c := key[i]
		if !isLetter(c) && !isDigit(c) && c != '-' {
			return false
		}
	}

	return true
}

func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// ParseTrailersFromMessage extracts AI trailers from a commit message
func ParseTrailersFromMessage(msg string) *Trailers {
	trailers := &Trailers{}

	scanner := bufio.NewScanner(strings.NewReader(msg))
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, TrailerSessionID+":") {
			trailers.SessionID = strings.TrimSpace(strings.TrimPrefix(line, TrailerSessionID+":"))
		} else if strings.HasPrefix(line, TrailerSlug+":") {
			trailers.Slug = strings.TrimSpace(strings.TrimPrefix(line, TrailerSlug+":"))
		} else if strings.HasPrefix(line, TrailerTool+":") {
			trailers.Tool = strings.TrimSpace(strings.TrimPrefix(line, TrailerTool+":"))
		}
	}

	return trailers
}

// HasAITrailers checks if a commit message has AI trailers
func HasAITrailers(msg string) bool {
	return strings.Contains(msg, TrailerSessionID+":")
}
