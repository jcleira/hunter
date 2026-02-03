package git

import (
	"strings"
	"testing"
)

func TestParseTrailersFromMessage(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected *Trailers
	}{
		{
			name: "message with trailers",
			message: `Add feature X

This is the body.

AI-Session: abc123
AI-Slug: tender-snacking-unicorn
AI-Tool: claude`,
			expected: &Trailers{
				SessionID: "abc123",
				Slug:      "tender-snacking-unicorn",
				Tool:      "claude",
			},
		},
		{
			name: "message without trailers",
			message: `Regular commit

No AI here.`,
			expected: &Trailers{},
		},
		{
			name: "message with only session",
			message: `Fix bug

AI-Session: xyz789`,
			expected: &Trailers{
				SessionID: "xyz789",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseTrailersFromMessage(tt.message)
			if result.SessionID != tt.expected.SessionID {
				t.Errorf("SessionID = %q, want %q", result.SessionID, tt.expected.SessionID)
			}
			if result.Slug != tt.expected.Slug {
				t.Errorf("Slug = %q, want %q", result.Slug, tt.expected.Slug)
			}
			if result.Tool != tt.expected.Tool {
				t.Errorf("Tool = %q, want %q", result.Tool, tt.expected.Tool)
			}
		})
	}
}

func TestAddTrailersToMessage(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		trailers *Trailers
		contains []string
	}{
		{
			name:    "add to simple message",
			message: "Add feature X",
			trailers: &Trailers{
				SessionID: "abc123",
				Slug:      "tender-unicorn",
			},
			contains: []string{"AI-Session: abc123", "AI-Slug: tender-unicorn"},
		},
		{
			name: "add to message with body",
			message: `Fix bug

This fixes the issue.`,
			trailers: &Trailers{
				SessionID: "xyz789",
			},
			contains: []string{"AI-Session: xyz789", "Fix bug", "This fixes the issue"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := addTrailersToMessage(tt.message, tt.trailers)
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("result should contain %q, got:\n%s", expected, result)
				}
			}
		})
	}
}

func TestIsTrailerLine(t *testing.T) {
	tests := []struct {
		line     string
		expected bool
	}{
		{"AI-Session: abc123", true},
		{"Co-Authored-By: Name <email>", true},
		{"Signed-off-by: Name", true},
		{"Regular line", false},
		{"No colon here", false},
		{"123-Invalid: starts with number", false},
		{": no key", false},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			result := isTrailerLine(tt.line)
			if result != tt.expected {
				t.Errorf("isTrailerLine(%q) = %v, want %v", tt.line, result, tt.expected)
			}
		})
	}
}

func TestHasAITrailers(t *testing.T) {
	tests := []struct {
		message  string
		expected bool
	}{
		{"Regular commit\n\nAI-Session: abc123", true},
		{"Regular commit without AI", false},
		{"AI-Session: in subject line", true},
	}

	for _, tt := range tests {
		t.Run(tt.message[:20], func(t *testing.T) {
			result := HasAITrailers(tt.message)
			if result != tt.expected {
				t.Errorf("HasAITrailers() = %v, want %v", result, tt.expected)
			}
		})
	}
}
