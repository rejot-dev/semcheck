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
				"main-title":    "# Main Title\n\nThis is the main content under the title.\n\n## Section One\n\nContent for section one.\nSome more content here.\n\n## Section Two\n\nContent for section two.\n\n### Subsection\n\nSubsection content.\n\n## Final Section\n\nFinal content.",
				"section-one":   "## Section One\n\nContent for section one.\nSome more content here.",
				"section-two":   "## Section Two\n\nContent for section two.\n\n### Subsection\n\nSubsection content.",
				"subsection":    "### Subsection\n\nSubsection content.",
				"final-section": "## Final Section\n\nFinal content.",
			},
		},
		{
			name: "headings with special characters",
			content: `# Title with Special & Characters!

Content here.

## Section: With Colon

More content.`,
			expected: map[string]string{
				"title-with-special-characters": "# Title with Special & Characters!\n\nContent here.\n\n## Section: With Colon\n\nMore content.",
				"section-with-colon":            "## Section: With Colon\n\nMore content.",
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
				"level-1":         "# Level 1\n\nContent at level 1.\n\n## Level 2\n\nContent at level 2.\n\n### Level 3\n\nContent at level 3.\n\n#### Level 4\n\nContent at level 4.\n\n## Another Level 2\n\nBack to level 2.",
				"level-2":         "## Level 2\n\nContent at level 2.\n\n### Level 3\n\nContent at level 3.\n\n#### Level 4\n\nContent at level 4.",
				"level-3":         "### Level 3\n\nContent at level 3.\n\n#### Level 4\n\nContent at level 4.",
				"level-4":         "#### Level 4\n\nContent at level 4.",
				"another-level-2": "## Another Level 2\n\nBack to level 2.",
			},
		},
		{
			name: "empty sections",
			content: `# First Section

## Empty Section

## Section with Content

This has content.`,
			expected: map[string]string{
				"first-section":        "# First Section\n\n## Empty Section\n\n## Section with Content\n\nThis has content.",
				"empty-section":        "## Empty Section",
				"section-with-content": "## Section with Content\n\nThis has content.",
			},
		},
		{
			name: "nested content inclusion",
			content: `# level 1
hey
## level 2
there`,
			expected: map[string]string{
				"level-1": "# level 1\nhey\n## level 2\nthere",
				"level-2": "## level 2\nthere",
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
				"test-section":   "# Test Section\n\nContent 1",
				"test-section-2": "# Test Section\n\nContent 2\n\n## Test Section\n\nContent 3",
				"test-section-3": "## Test Section\n\nContent 3",
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
			result := textToAnchor(tt.input)
			if result != tt.expected {
				t.Errorf("generateAnchor(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestStructuredDocument_GetSubsection(t *testing.T) {
	doc := CollectedDocument{
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
			result := doc.GetAnchoredSection(tt.anchor)
			if result != tt.expected {
				t.Errorf("GetAnchoredSection(%q) = %q, want %q", tt.anchor, result, tt.expected)
			}
		})
	}
}
