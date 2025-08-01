package processor

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// Compile regex once at package level for efficiency
var anchorRegex = regexp.MustCompile(`[^a-z0-9]+`)

type markdownParser struct{}

func NewMarkdownParser() DocumentParser {
	return &markdownParser{}
}

func (p *markdownParser) Parse(content []byte) (StructuredDocument, error) {
	anchors := make(map[string]string)

	// Parse markdown using goldmark
	md := goldmark.New()
	doc := md.Parser().Parse(text.NewReader(content))

	// Collect all headings with their positions
	var headings []headingInfo
	err := ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering && node.Kind() == ast.KindHeading {
			heading := node.(*ast.Heading)
			headingText := extractNodeText(heading, content)
			anchor := generateAnchor(headingText)

			headings = append(headings, headingInfo{
				node:   node,
				anchor: anchor,
				level:  heading.Level,
			})
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return StructuredDocument{}, err
	}

	// Extract content for each heading
	anchorCounts := make(map[string]int)
	for i, heading := range headings {
		// Handle duplicate anchors by appending a number
		anchor := heading.anchor
		if count, exists := anchorCounts[heading.anchor]; exists {
			anchor = fmt.Sprintf("%s-%d", heading.anchor, count+1)
			anchorCounts[heading.anchor]++
		} else {
			anchorCounts[heading.anchor] = 1
		}

		sectionContent := extractSectionContentBetweenHeadings(heading.node, getNextSameLevelOrHigher(headings, i), content)
		anchors[anchor] = sectionContent
	}

	return StructuredDocument{
		Type:    Markdown,
		content: content,
		anchors: anchors,
	}, nil
}

type headingInfo struct {
	node   ast.Node
	anchor string
	level  int
}

func extractNodeText(node ast.Node, source []byte) string {
	var sb strings.Builder
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Kind() == ast.KindText {
			if textNode, ok := child.(*ast.Text); ok {
				sb.Write(textNode.Value(source))
			}
		}
	}
	return sb.String()
}

func generateAnchor(text string) string {
	// Convert to lowercase
	anchor := strings.ToLower(text)

	// Replace spaces and other characters with hyphens
	anchor = anchorRegex.ReplaceAllString(anchor, "-")

	// Remove leading/trailing hyphens
	anchor = strings.Trim(anchor, "-")

	return anchor
}

func getNextSameLevelOrHigher(headings []headingInfo, currentIndex int) ast.Node {
	if currentIndex >= len(headings)-1 {
		return nil
	}

	currentLevel := headings[currentIndex].level

	// Find the next heading at the same level or higher (lower number)
	// This ensures sections include all nested content
	for i := currentIndex + 1; i < len(headings); i++ {
		if headings[i].level <= currentLevel {
			return headings[i].node
		}
	}
	return nil
}

func extractSectionContentBetweenHeadings(startHeading, endHeading ast.Node, source []byte) string {
	var content bytes.Buffer

	// Start from the node after the heading
	current := startHeading.NextSibling()

	for current != nil && current != endHeading {
		// Include ALL content - both headings and non-heading content
		nodeContent := extractFullNodeText(current, source)
		if nodeContent != "" {
			if content.Len() > 0 {
				content.WriteString("\n")
			}
			content.WriteString(nodeContent)
		}
		current = current.NextSibling()
	}

	return strings.TrimSpace(content.String())
}

func extractFullNodeText(node ast.Node, source []byte) string {
	var parts []string

	if node.Kind() == ast.KindParagraph {
		// For paragraphs, collect text nodes and join with newlines to preserve line breaks
		var texts []string
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			if child.Kind() == ast.KindText {
				if textNode, ok := child.(*ast.Text); ok {
					texts = append(texts, string(textNode.Value(source)))
				}
			}
		}
		if len(texts) > 0 {
			parts = append(parts, strings.Join(texts, "\n"))
		}
	} else {
		// For other nodes, walk and collect text
		err := ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
			if entering && n.Kind() == ast.KindText {
				if textNode, ok := n.(*ast.Text); ok {
					parts = append(parts, string(textNode.Value(source)))
				}
			}
			return ast.WalkContinue, nil
		})
		// Note: ast.Walk in goldmark typically doesn't return errors for simple traversal
		_ = err
	}

	return strings.TrimSpace(strings.Join(parts, ""))
}
