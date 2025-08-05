package processor

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

type htmlParser struct{}

func NewHTMLParser() DocumentParser {
	return &htmlParser{}
}

func (p *htmlParser) Parse(content []byte) (CollectedDocument, error) {
	anchors := make(map[string]string)
	source := string(content)

	doc, err := html.Parse(strings.NewReader(source))
	if err != nil {
		return CollectedDocument{}, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Filter out unwanted elements before processing
	filterUnwantedElements(doc)

	// Traverse the HTML tree to find all elements with IDs or elements with names
	// semcheck:url(https://html.spec.whatwg.org/multipage/browsing-the-web.html#select-the-indicated-part)
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode {
			// Check for ID attribute (any element can have an ID)
			for _, attr := range n.Attr {
				if attr.Key == "id" && attr.Val != "" {
					// Get contextual content based on element type
					htmlContent := getContextualContent(n)
					anchors[attr.Val] = htmlContent
					break
				}
			}

			// Check for name attribute on anchor elements
			if n.Data == "a" {
				for _, attr := range n.Attr {
					if attr.Key == "name" && attr.Val != "" {
						// Get contextual content for anchor elements
						htmlContent := getContextualContent(n)
						anchors[attr.Val] = htmlContent
						break
					}
				}
			}
		}

		// Traverse children
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(doc)

	return CollectedDocument{
		Type:    HTML,
		content: content,
		anchors: anchors,
	}, nil
}

// renderNode converts an HTML node and its children back to HTML string
func renderNode(n *html.Node) string {
	var buf strings.Builder
	_ = html.Render(&buf, n)
	return buf.String()
}

// isUnwantedElement checks if the node is an element that should be filtered out
func isUnwantedElement(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}

	unwantedTags := map[string]bool{
		"script":   true,
		"style":    true,
		"svg":      true,
		"canvas":   true,
		"audio":    true,
		"video":    true,
		"embed":    true,
		"object":   true,
		"iframe":   true,
		"noscript": true,
		"form":     true,
		"input":    true,
		"button":   true,
		"select":   true,
		"textarea": true,
		"meta":     true,
		"link":     true,
		"base":     true,
		"title":    true,
		"head":     true,
	}

	return unwantedTags[n.Data]
}

// isAllowedAttribute checks if an attribute should be preserved
func isAllowedAttribute(attrKey string) bool {
	allowedAttrs := map[string]bool{
		// Essential for anchoring and navigation
		"id":   true,
		"name": true,
		"href": true,
	}

	return allowedAttrs[attrKey]
}

// filterAttributes removes unwanted attributes from an element, keeping only allowed ones
func filterAttributes(n *html.Node) {
	if n.Type != html.ElementNode {
		return
	}

	var filteredAttrs []html.Attribute
	for _, attr := range n.Attr {
		if isAllowedAttribute(attr.Key) {
			filteredAttrs = append(filteredAttrs, attr)
		}
	}
	n.Attr = filteredAttrs
}

// filterUnwantedElements recursively removes unwanted elements and attributes from the HTML tree
func filterUnwantedElements(n *html.Node) {
	child := n.FirstChild
	for child != nil {
		next := child.NextSibling

		if isUnwantedElement(child) {
			// Remove the unwanted element
			n.RemoveChild(child)
		} else {
			// Filter attributes from this element
			filterAttributes(child)
			// Recursively filter children
			filterUnwantedElements(child)
		}

		child = next
	}
}

// isHeadingElement checks if the node is a heading element (h1-h6)
func isHeadingElement(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	return n.Data == "h1" || n.Data == "h2" || n.Data == "h3" ||
		n.Data == "h4" || n.Data == "h5" || n.Data == "h6"
}

// getHeadingLevel returns the numeric level of a heading (1-6), or 0 if not a heading
func getHeadingLevel(n *html.Node) int {
	if !isHeadingElement(n) {
		return 0
	}
	switch n.Data {
	case "h1":
		return 1
	case "h2":
		return 2
	case "h3":
		return 3
	case "h4":
		return 4
	case "h5":
		return 5
	case "h6":
		return 6
	default:
		return 0
	}
}

// isSectioningElement checks if the node is a sectioning element
func isSectioningElement(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	return n.Data == "section" || n.Data == "article" ||
		n.Data == "aside" || n.Data == "nav" ||
		n.Data == "main" || n.Data == "body"
}

// findNearestSectioningContainer traverses up the DOM to find the nearest sectioning container
func findNearestSectioningContainer(n *html.Node) *html.Node {
	current := n.Parent
	for current != nil {
		if isSectioningElement(current) {
			return current
		}
		current = current.Parent
	}
	return nil // Should not happen since body is always a sectioning element
}

// collectContentFollowingElement collects content that follows the target element within its sectioning container
func collectContentFollowingElement(targetElement *html.Node) string {
	// Start with the target element itself
	var contentNodes []*html.Node
	contentNodes = append(contentNodes, targetElement)

	// Find the nearest sectioning container
	sectionContainer := findNearestSectioningContainer(targetElement)
	if sectionContainer == nil {
		return renderNode(targetElement)
	}

	// Find the parent element that is a direct child of the sectioning container
	var parentInSection *html.Node
	current := targetElement.Parent
	for current != nil && current.Parent != sectionContainer {
		current = current.Parent
	}
	parentInSection = current

	if parentInSection == nil {
		return renderNode(targetElement)
	}

	// Find the heading level that defines this section (if any)
	var sectionHeadingLevel int
	for sibling := sectionContainer.FirstChild; sibling != nil; sibling = sibling.NextSibling {
		if sibling == parentInSection {
			break
		}
		if isHeadingElement(sibling) {
			sectionHeadingLevel = getHeadingLevel(sibling)
		}
	}

	// Collect all sibling elements that come after the parent element within the section
	collecting := false
	for sibling := sectionContainer.FirstChild; sibling != nil; sibling = sibling.NextSibling {
		if sibling == parentInSection {
			collecting = true
			continue // We already have content from this element (the target)
		}

		if collecting {
			// Stop if we encounter another sectioning element
			if isSectioningElement(sibling) {
				break
			}

			// Stop if we encounter a heading of equal or higher level than the section heading
			if isHeadingElement(sibling) && sectionHeadingLevel > 0 {
				siblingLevel := getHeadingLevel(sibling)
				if siblingLevel <= sectionHeadingLevel {
					break
				}
			}

			contentNodes = append(contentNodes, sibling)
		}
	}

	// Render all collected nodes
	var buf strings.Builder
	for _, node := range contentNodes {
		_ = html.Render(&buf, node)
	}

	return buf.String()
}

// collectSectionContent collects content starting from a heading element until section boundary
func collectSectionContent(headingNode *html.Node) string {
	if !isHeadingElement(headingNode) {
		return renderNode(headingNode)
	}

	headingLevel := getHeadingLevel(headingNode)
	var contentNodes []*html.Node
	contentNodes = append(contentNodes, headingNode)

	// Collect all following sibling nodes until we hit a section boundary
	for sibling := headingNode.NextSibling; sibling != nil; sibling = sibling.NextSibling {
		// Skip text nodes that are just whitespace
		if sibling.Type == html.TextNode && strings.TrimSpace(sibling.Data) == "" {
			continue
		}

		// If we encounter another heading of equal or higher level, stop
		if isHeadingElement(sibling) {
			siblingLevel := getHeadingLevel(sibling)
			if siblingLevel <= headingLevel {
				break
			}
		}

		// If we encounter a sectioning element, stop
		if isSectioningElement(sibling) {
			break
		}

		contentNodes = append(contentNodes, sibling)
	}

	// Render all collected nodes
	var buf strings.Builder
	for _, node := range contentNodes {
		_ = html.Render(&buf, node)
	}

	return buf.String()
}

// getContextualContent determines what content to extract based on element type
func getContextualContent(n *html.Node) string {
	// For heading elements, collect the entire section
	if isHeadingElement(n) {
		return collectSectionContent(n)
	}

	// For sectioning elements, capture the entire container
	if isSectioningElement(n) {
		return renderNode(n)
	}

	// For all other elements (non-heading, non-sectioning), collect content following the element within its section
	return collectContentFollowingElement(n)
}
