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
	ErrorLocalFileNoExtension                     = errors.New("error collecting document, local file has no extension")
	ErrorAnchorNotSupportedOnUnstructuredDocument = errors.New("error collecting document, anchor not supported on unstructured document")
	ErrorParsing                                  = errors.New("error parsing document")
)

// TODO: Rename, if type == TXT then it is actually unstructured
type StructuredDocument struct {
	Type    DocumentType
	content []byte
	anchors map[string]string
}

func UnstructuredDocument(content []byte) StructuredDocument {
	return StructuredDocument{
		Type:    TXT,
		content: content,
		anchors: make(map[string]string),
	}
}

func (s *StructuredDocument) GetSubsection(anchor string) string {
	return s.anchors[anchor]
}

type DocumentParser interface {
	Parse(content []byte) (StructuredDocument, error)
}

func collectLocalFile(path string) ([]byte, DocumentType, error) {
	// Read file content
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	fileExtension := filepath.Ext(path)

	if fileExtension == "" {
		return nil, UNKNOWN, ErrorLocalFileNoExtension
	}

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

func CollectDocument(path string) (structuredDoc StructuredDocument, anchor string, err error) {
	parsedUrl, err := url.Parse(path)

	if err != nil {
		return StructuredDocument{}, "", ErrorCollectingBadURL
	}

	anchor = parsedUrl.Fragment

	var content []byte
	var docType DocumentType

	if parsedUrl.Scheme == "" || parsedUrl.Scheme == "file" {
		log.Debug("Collecting local file", "path", path)
		content, docType, err = collectLocalFile(path)
	} else {
		log.Debug("Collecting remote file", "path", path)
		content, docType, err = collectRemoteFile(parsedUrl)
	}

	if err != nil {
		return StructuredDocument{}, anchor, err
	}

	var parser DocumentParser

	switch docType {
	case Markdown:
		parser = NewMarkdownParser()
	case TXT:
		if anchor != "" {
			return StructuredDocument{}, anchor, ErrorAnchorNotSupportedOnUnstructuredDocument
		}
		return UnstructuredDocument(content), "", nil
	case UNKNOWN:
		// Consider to return an unstructured document as fallback
		return StructuredDocument{}, anchor, ErrorCollectingUnknownDocumentType
	default:
		return StructuredDocument{}, anchor, ErrorCollectingNoDocumentParser
	}

	structuredDoc, err = parser.Parse(content)

	return structuredDoc, anchor, err
}
