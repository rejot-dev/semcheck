package processor

import (
	"testing"
)

func TestMarkdownParser_Parse(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected map[string]string
	}{
		{
			name: "simple headings with content",
			content: `# Main Title

This is the main content under the title.

## Section One

Content for section one.
Some more content here.

## Section Two

Content for section two.

### Subsection

Subsection content.

## Final Section

Final content.`,
			expected: map[string]string{
				"main-title":    "This is the main content under the title.\nSection One\nContent for section one.\nSome more content here.\nSection Two\nContent for section two.\nSubsection\nSubsection content.\nFinal Section\nFinal content.",
				"section-one":   "Content for section one.\nSome more content here.",
				"section-two":   "Content for section two.\nSubsection\nSubsection content.",
				"subsection":    "Subsection content.",
				"final-section": "Final content.",
			},
		},
		{
			name: "headings with special characters",
			content: `# Title with Special & Characters!

Content here.

## Section: With Colon

More content.`,
			expected: map[string]string{
				"title-with-special-characters": "Content here.\nSection: With Colon\nMore content.",
				"section-with-colon":            "More content.",
			},
		},
		{
			name: "nested heading levels",
			content: `# Level 1

Content at level 1.

## Level 2

Content at level 2.

### Level 3

Content at level 3.

#### Level 4

Content at level 4.

## Another Level 2

Back to level 2.`,
			expected: map[string]string{
				"level-1":         "Content at level 1.\nLevel 2\nContent at level 2.\nLevel 3\nContent at level 3.\nLevel 4\nContent at level 4.\nAnother Level 2\nBack to level 2.",
				"level-2":         "Content at level 2.\nLevel 3\nContent at level 3.\nLevel 4\nContent at level 4.",
				"level-3":         "Content at level 3.\nLevel 4\nContent at level 4.",
				"level-4":         "Content at level 4.",
				"another-level-2": "Back to level 2.",
			},
		},
		{
			name: "empty sections",
			content: `# First Section

## Empty Section

## Section with Content

This has content.`,
			expected: map[string]string{
				"first-section":        "Empty Section\nSection with Content\nThis has content.",
				"empty-section":        "",
				"section-with-content": "This has content.",
			},
		},
		{
			name: "nested content inclusion",
			content: `# level 1
hey
## level 2
there`,
			expected: map[string]string{
				"level-1": "hey\nlevel 2\nthere",
				"level-2": "there",
			},
		},
		{
			name: "duplicate anchor handling",
			content: `# Test Section

Content 1

# Test Section

Content 2

## Test Section

Content 3`,
			expected: map[string]string{
				"test-section":   "Content 1",
				"test-section-2": "Content 2\nTest Section\nContent 3",
				"test-section-3": "Content 3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewMarkdownParser()
			result, err := parser.Parse([]byte(tt.content))

			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if result.Type != Markdown {
				t.Errorf("Parse() Type = %v, want %v", result.Type, Markdown)
			}

			if len(result.anchors) != len(tt.expected) {
				t.Errorf("Parse() anchors count = %v, want %v", len(result.anchors), len(tt.expected))
			}

			for expectedAnchor, expectedContent := range tt.expected {
				actualContent, exists := result.anchors[expectedAnchor]
				if !exists {
					t.Errorf("Parse() missing anchor %q", expectedAnchor)
					continue
				}

				if actualContent != expectedContent {
					t.Errorf("Parse() anchor %q content = %q, want %q", expectedAnchor, actualContent, expectedContent)
				}
			}
		})
	}
}

func TestGenerateAnchor(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Simple Title", "simple-title"},
		{"Title with Special & Characters!", "title-with-special-characters"},
		{"  Spaces   Around  ", "spaces-around"},
		{"UPPERCASE", "uppercase"},
		{"Title: With Colon", "title-with-colon"},
		{"Numbers 123 Test", "numbers-123-test"},
		{"Multiple---Dashes", "multiple-dashes"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := generateAnchor(tt.input)
			if result != tt.expected {
				t.Errorf("generateAnchor(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestStructuredDocument_GetSubsection(t *testing.T) {
	doc := StructuredDocument{
		Type:    Markdown,
		content: []byte("test content"),
		anchors: map[string]string{
			"section-one": "Content one",
			"section-two": "Content two",
		},
	}

	tests := []struct {
		anchor   string
		expected string
	}{
		{"section-one", "Content one"},
		{"section-two", "Content two"},
		{"nonexistent", ""},
	}

	for _, tt := range tests {
		t.Run(tt.anchor, func(t *testing.T) {
			result := doc.GetSubsection(tt.anchor)
			if result != tt.expected {
				t.Errorf("GetSubsection(%q) = %q, want %q", tt.anchor, result, tt.expected)
			}
		})
	}
}
