package processor

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/log"
)

func TestDocumentCollection(t *testing.T) {
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
		structuredDoc, anchor, err := CollectDocument("https://www.rfc-editor.org/rfc/rfc6762.txt")
		if err != nil {
			t.Fatal(err)
		}

		if structuredDoc.Type != TXT {
			t.Errorf("Expected docType txt, got %s", structuredDoc.Type)
		}
		if anchor != "" {
			t.Errorf("Expected anchor to be empty, got %s", anchor)
		}
	})

	t.Run("collect unstructured document with anchor", func(t *testing.T) {
		_, anchor, err := CollectDocument("https://www.rfc-editor.org/rfc/rfc6762.txt#section-1")
		if anchor != "section-1" {
			t.Errorf("Expected anchor to be section-1, got %s", anchor)
		}

		if err != ErrorAnchorNotSupportedOnUnstructuredDocument {
			t.Errorf("Expected %v, got %v", ErrorAnchorNotSupportedOnUnstructuredDocument, err)
		}
	})

	t.Run("collect remote markdown", func(t *testing.T) {
		doc, anchor, err := CollectDocument("https://raw.githubusercontent.com/rejot-dev/semcheck/4d634dc6bfd7b5a00b98b2f862f2fd567b08919b/specs/semcheck.md")

		if err != nil {
			t.Fatal(err)
		}

		if anchor != "" {
			t.Errorf("Expected anchor to be empty, got %s", anchor)
		}

		if doc.Type != Markdown {
			t.Errorf("Expected docType md, got %s", doc.Type)
		}
	})

	t.Run("collect document with anchor", func(t *testing.T) {
		_, _, err := CollectDocument("https://semcheck.ai#my-anchor")
		// TODO: add test once implemented
		if err != ErrorCollectingNoDocumentParser {
			t.Errorf("Expected %v, got %v", ErrorCollectingNoDocumentParser, err)
		}
	})
}
