package processor

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

type DocumentType string

const (
	Markdown = "md"
	HTML     = "html"
	TXT      = "txt"
	PDF      = "pdf"
	UNKNOWN  = "unkown"
)

var (
	ErrorCollectingBadURL                         = errors.New("error collecting document, bad URL")
	ErrorCollectingUnknownDocumentType            = errors.New("error collecting document, unknown document type")
	ErrorCollectingNoDocumentParser               = errors.New("error collecting document, no document parser for document type")
	ErrorAnchorNotSupportedOnUnstructuredDocument = errors.New("error collecting document, fragment or anchor not supported on this document")
	ErrorParsing                                  = errors.New("error parsing document")
)

type CollectedDocument struct {
	Type    DocumentType
	content []byte
	anchors map[string]string
}

func textToAnchor(text string) string {
	// Convert to lowercase
	anchor := strings.ToLower(text)

	// Replace spaces and other characters with hyphens
	anchor = anchorRegex.ReplaceAllString(anchor, "-")

	// Remove leading/trailing hyphens
	anchor = strings.Trim(anchor, "-")

	return anchor
}

func UnstructuredDocument(content []byte) CollectedDocument {
	return CollectedDocument{
		Type:    TXT,
		content: content,
		anchors: make(map[string]string),
	}
}

func (s *CollectedDocument) GetAnchoredSection(anchor string) (string, error) {
	if !s.IsStructuredDocument() {
		return "", ErrorAnchorNotSupportedOnUnstructuredDocument
	}
	return s.anchors[textToAnchor(anchor)], nil
}

func (s *CollectedDocument) IsStructuredDocument() bool {
	return s.Type == Markdown || s.Type == HTML
}

type DocumentParser interface {
	Parse(content []byte) (CollectedDocument, error)
}

func collectLocalFile(path string) ([]byte, DocumentType, error) {
	// Read file content
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	fileExtension := filepath.Ext(path)
	return content, extensionToDocumentType(fileExtension), nil
}

func extensionToDocumentType(ext string) DocumentType {
	switch ext {
	case ".md":
		return Markdown
	case ".html":
		return HTML
	case ".txt":
		return TXT
	case ".pdf":
		return PDF
	default:
		return UNKNOWN
	}
}

func mimeToDocumentType(headerString string) DocumentType {
	if headerString == "" {
		return UNKNOWN
	}

	// mime-type = type "/" [tree "."] subtype ["+" suffix]* [";" parameter];
	// Example: text/plain; charset=utf-8
	headerParts := strings.Split(headerString, ";")

	log.Debug("Mimetype received", "mime", headerParts[0])

	switch headerParts[0] {
	case "text/html":
		return HTML
	case "text/plain":
		return TXT
	case "application/pdf":
		return PDF
	default:
		return UNKNOWN
	}
}

func collectRemoteFile(url *url.URL) ([]byte, DocumentType, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url.String())

	if err != nil {
		return nil, UNKNOWN, fmt.Errorf("failed to fetch URL %s: %w", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, UNKNOWN, fmt.Errorf("failed to fetch URL %s: HTTP %d %s", url, resp.StatusCode, resp.Status)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, UNKNOWN, fmt.Errorf("failed to read content from URL %s: %w", url, err)
	}

	docType := mimeToDocumentType(resp.Header.Get("Content-Type"))

	if docType == TXT {
		// Markdown files will have mimeType text/plain, see if we can detect from the extension
		// if it's actually markdown
		if url.Path != "" && strings.HasSuffix(url.Path, ".md") {
			docType = Markdown
		}
	}

	return content, docType, nil
}

func CollectDocument(url *url.URL) (CollectedDocument, error) {
	var content []byte
	var docType DocumentType
	var err error

	if url.Scheme == "" || url.Scheme == "file" {
		log.Debug("Collecting local file", "url", url)
		content, docType, err = collectLocalFile(url.Path)
	} else {
		log.Debug("Collecting remote file", "url", url)
		content, docType, err = collectRemoteFile(url)
	}

	if err != nil {
		return CollectedDocument{}, err
	}

	var parser DocumentParser

	switch docType {
	case Markdown:
		parser = NewMarkdownParser()
	case HTML:
		parser = NewHTMLParser()
	case TXT:
		return UnstructuredDocument(content), nil
	case UNKNOWN:
		// Return an unstructured document as fallback
		return UnstructuredDocument(content), nil
	default:
		return CollectedDocument{}, ErrorCollectingNoDocumentParser
	}

	structuredDoc, err := parser.Parse(content)

	return structuredDoc, err
}

type DocumentCollection struct {
	DocumentCache map[string]CollectedDocument
}

func NewDocumentCollection() DocumentCollection {
	dc := DocumentCollection{
		DocumentCache: make(map[string]CollectedDocument),
	}
	return dc
}

func (dc *DocumentCollection) cacheKey(url *url.URL) string {
	var remoteLocal string
	if url.Scheme == "" || url.Scheme == "file" {
		remoteLocal = "local"
	} else {
		remoteLocal = "remote"
	}

	if url.Path == "" {
		url.Path = "/"
	}

	return fmt.Sprintf("%s-%s%s", remoteLocal, url.Host, url.Path)

}

func (dc *DocumentCollection) GetDocument(path string) (string, error) {
	url, err := url.Parse(path)

	if err != nil {
		return "", ErrorCollectingBadURL
	}

	anchor := url.Fragment

	key := dc.cacheKey(url)
	var retrievedDoc CollectedDocument

	if doc, ok := dc.DocumentCache[key]; ok {
		retrievedDoc = doc
	} else {
		doc, err := CollectDocument(url)
		if err != nil {
			return "", err
		}

		retrievedDoc = doc
		dc.DocumentCache[key] = doc
	}

	if anchor != "" {
		return retrievedDoc.GetAnchoredSection(anchor)
	}

	return string(retrievedDoc.content), nil
}
