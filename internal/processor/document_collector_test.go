package processor

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/log"
)

func TestDocumentCollector(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("local file collection", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "spec.md")

		content := []byte("# Spec\n hey")
		err := os.WriteFile(testFile, content, 0644)
		if err != nil {
			t.Fatal(err)
		}

		readContent, docType, err := collectLocalFile(string(testFile))
		if err != nil {
			t.Fatal(err)
		}
		if string(readContent) != string(content) {
			t.Errorf("Expected content %s, got %s", string(content), string(readContent))
		}
		if docType != Markdown {
			t.Errorf("Expected docType markdown, got %s", docType)
		}
	})

	t.Run("remote collection", func(t *testing.T) {
		parsedUrl, _ := url.Parse("https://semcheck.ai")
		_, docType, err := collectRemoteFile(parsedUrl)
		if err != nil {
			t.Fatal(err)
		}

		if docType != HTML {
			t.Errorf("Expected docType html, got %s", docType)
		}
	})

	t.Run("collect unstructured document", func(t *testing.T) {
		parsedURL, _ := url.Parse("https://www.rfc-editor.org/rfc/rfc6762.txt")
		structuredDoc, err := CollectDocument(parsedURL)
		if err != nil {
			t.Fatal(err)
		}

		if structuredDoc.Type != TXT {
			t.Errorf("Expected docType txt, got %s", structuredDoc.Type)
		}
	})

	t.Run("collect remote markdown", func(t *testing.T) {
		parsedURL, _ := url.Parse("https://raw.githubusercontent.com/rejot-dev/semcheck/4d634dc6bfd7b5a00b98b2f862f2fd567b08919b/specs/semcheck.md")
		doc, err := CollectDocument(parsedURL)

		if err != nil {
			t.Fatal(err)
		}

		if doc.Type != Markdown {
			t.Errorf("Expected docType md, got %s", doc.Type)
		}
	})

	t.Run("collect document with anchor", func(t *testing.T) {
		parsedURL, _ := url.Parse("https://semcheck.ai#my-anchor")
		_, err := CollectDocument(parsedURL)
		// TODO: add test once implemented
		if err != ErrorCollectingNoDocumentParser {
			t.Errorf("Expected %v, got %v", ErrorCollectingNoDocumentParser, err)
		}
	})
}

func TestTextToAnchor(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello-world"},
		{"Section 1.1", "section-1-1"},
		{"section @ level 2", "section-level-2"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := textToAnchor(tt.input)
			if result != tt.expected {
				t.Errorf("textToAnchor(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDocumentCollection(t *testing.T) {

	dc := NewDocumentCollection()

	cacheKeyTests := []struct {
		input    string
		expected string
	}{
		{"https://semcheck.ai", "remote-semcheck.ai/"},
		{"http://semcheck.ai", "remote-semcheck.ai/"},
		{"https://semcheck.ai/", "remote-semcheck.ai/"},
		{"https://semcheck.ai/#section", "remote-semcheck.ai/"},
		{"./specs/my-spec.md", "local-./specs/my-spec.md"},
		{"../specs/my-spec.md", "local-../specs/my-spec.md"},
		{"../specs/my-spec.md#section-10", "local-../specs/my-spec.md"},
		{"https://www.rfc-editor.org/rfc/rfc6762.txt#section-1", "remote-www.rfc-editor.org/rfc/rfc6762.txt"},
	}

	for _, tt := range cacheKeyTests {
		t.Run("cache key "+tt.input, func(t *testing.T) {
			url, _ := url.Parse(tt.input)
			key := dc.cacheKey(url)
			if key != tt.expected {
				t.Errorf("cacheKey(%q) = %q, want %q", tt.input, key, tt.expected)
			}
		})
	}

	t.Run("document collection", func(t *testing.T) {
		content, err := dc.GetDocument("https://www.rfc-editor.org/rfc/rfc6762.txt")

		if err != nil {
			t.Errorf("GetDocument() error = %v", err)
		}

		if len(content) < 1 {
			t.Errorf("GetDocument() content length = %d, want > 0", len(content))
		}
	})
	t.Run("anchors error on unstructureddocument collection", func(t *testing.T) {
		_, err := dc.GetDocument("https://www.rfc-editor.org/rfc/rfc6762.txt#section-1")
		if err != ErrorAnchorNotSupportedOnUnstructuredDocument {
			t.Errorf("GetDocument() error is not the one expected = %v", err)
		}
	})
}
