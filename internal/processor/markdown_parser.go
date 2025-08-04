package processor

import (
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

func (p *markdownParser) Parse(content []byte) (CollectedDocument, error) {
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
			anchor := textToAnchor(headingText)

			headings = append(headings, headingInfo{
				node:   node,
				anchor: anchor,
				level:  heading.Level,
			})
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return CollectedDocument{}, err
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

	return CollectedDocument{
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
	// Convert source to string for easier string manipulation
	sourceStr := string(source)

	// Get the raw start and end positions from the AST
	startLine := int(startHeading.(*ast.Heading).Lines().At(0).Start)

	// Find the beginning of the line (includes the markdown heading symbols)
	startPos := startLine
	for startPos > 0 && sourceStr[startPos-1] != '\n' {
		startPos--
	}

	var endPos int
	if endHeading != nil {
		// Get position of the next heading line start
		endLine := int(endHeading.(*ast.Heading).Lines().At(0).Start)
		endPos = endLine
		// Find the beginning of that line
		for endPos > 0 && sourceStr[endPos-1] != '\n' {
			endPos--
		}
	} else {
		// If no next heading, go to end of document
		endPos = len(sourceStr)
	}

	// Extract raw markdown content from source, preserving original formatting
	rawContent := sourceStr[startPos:endPos]

	return strings.TrimSpace(rawContent)
}
