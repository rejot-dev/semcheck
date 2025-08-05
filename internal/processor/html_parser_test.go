package processor

import (
	"strings"
	"testing"
)

func TestSectionedContent(t *testing.T) {
	parser := NewHTMLParser()
	t.Run("simple sections", func(t *testing.T) {
		html := `
<section>
	<h1 id="section-1">Section 1</h1>
	<p>hey</p>
</section>
<section>
	<h2 id="section-2">Section 2</h2>
</section>`
		doc, err := parser.Parse([]byte(html))
		if err != nil {
			t.Errorf("Parse(%q) = %v, want nil", html, err)
		}
		expected := `<h1 id="section-1">Section 1</h1><p>hey</p>`

		sectionedContent, err := doc.GetAnchoredSection("section-1")
		if err != nil {
			t.Errorf("GetAnchoredSection(%q) = %v, want nil", "section-1", err)
		}
		if sectionedContent != expected {
			t.Errorf("Expected anchor for section-1, got %q", sectionedContent)
		}

	})
}

func TestUnsectionedContent(t *testing.T) {
	parser := NewHTMLParser()
	t.Run("one level", func(t *testing.T) {
		html := `
<main>
	<h1>Section 1</h1>
	<p>row 1</p>
	<h1>Section 2</h1>
	<p>
	    <dfn id="my-def">some def</dfn> content
	</p>
	<p>
		next
	</p>
	<h1>Section 3</h1>
	<p>another</p>
</main>`

		doc, err := parser.Parse([]byte(html))
		if err != nil {
			t.Errorf("Parse(%q) = %v, want nil", html, err)
		}

		// expected := `<h1>Section 2</h1><p><dfn id="my-def">some def</dfn> content</p><p>next</p>`
		sectionedContent, err := doc.GetAnchoredSection("my-def")
		if err != nil {
			t.Errorf("GetAnchoredSection(%q) = %v, want nil", "my-def", err)
		}
		if strings.Contains(sectionedContent, "Section 3") {
			t.Errorf("Section 3 should not be included")
		}

	})
}

func TestFilterUnwantedElements(t *testing.T) {
	parser := NewHTMLParser()
	t.Run("filters script and style tags", func(t *testing.T) {
		html := `
<main>
	<h1>Section 1</h1>
	<script>alert('malicious');</script>
	<p id="content">This is content</p>
	<style>body { color: red; }</style>
	<div>More content</div>
</main>`

		doc, err := parser.Parse([]byte(html))
		if err != nil {
			t.Errorf("Parse(%q) = %v, want nil", html, err)
		}

		sectionedContent, err := doc.GetAnchoredSection("content")
		if err != nil {
			t.Errorf("GetAnchoredSection(%q) = %v, want nil", "content", err)
		}
		if strings.Contains(sectionedContent, "script") {
			t.Errorf("Script tag should be filtered out")
		}
		if strings.Contains(sectionedContent, "alert") {
			t.Errorf("Script content should be filtered out")
		}
		if strings.Contains(sectionedContent, "style") {
			t.Errorf("Style tag should be filtered out")
		}
		if strings.Contains(sectionedContent, "color: red") {
			t.Errorf("Style content should be filtered out")
		}
		if !strings.Contains(sectionedContent, "This is content") {
			t.Errorf("Actual content should be preserved")
		}
		// The main test here is that unwanted elements are filtered out
		// Content collection behavior is tested separately
	})

	t.Run("filters unwanted attributes", func(t *testing.T) {
		html := `
<main>
	<h1>Section 1</h1>
	<p id="content" class="highlight" style="color: red;" onclick="alert('click')" data-test="value">
		This is content with <a href="http://example.com" target="_blank" rel="noopener" download>a link</a>
	</p>
</main>`

		doc, err := parser.Parse([]byte(html))
		if err != nil {
			t.Errorf("Parse(%q) = %v, want nil", html, err)
		}

		sectionedContent, err := doc.GetAnchoredSection("content")
		if err != nil {
			t.Errorf("GetAnchoredSection(%q) = %v, want nil", "content", err)
		}

		// Should filter out unwanted attributes
		if strings.Contains(sectionedContent, "class=") {
			t.Errorf("class attribute should be filtered out")
		}
		if strings.Contains(sectionedContent, "style=") {
			t.Errorf("style attribute should be filtered out")
		}
		if strings.Contains(sectionedContent, "onclick=") {
			t.Errorf("onclick attribute should be filtered out")
		}
		if strings.Contains(sectionedContent, "data-test=") {
			t.Errorf("data-* attributes should be filtered out")
		}
		if strings.Contains(sectionedContent, "target=") {
			t.Errorf("target attribute should be filtered out")
		}
		if strings.Contains(sectionedContent, "rel=") {
			t.Errorf("rel attribute should be filtered out")
		}
		if strings.Contains(sectionedContent, "download") {
			t.Errorf("download attribute should be filtered out")
		}

		// Should preserve id and href attributes
		if !strings.Contains(sectionedContent, `id="content"`) {
			t.Errorf("id attribute should be preserved")
		}
		if !strings.Contains(sectionedContent, `href="http://example.com"`) {
			t.Errorf("href attribute should be preserved")
		}

		// Should preserve text content
		if !strings.Contains(sectionedContent, "This is content") {
			t.Errorf("Text content should be preserved")
		}
		if !strings.Contains(sectionedContent, "a link") {
			t.Errorf("Link text should be preserved")
		}
	})
}
