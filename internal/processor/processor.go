package processor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ledongthuc/pdf"
)

type DocumentProcessor interface {
	ExtractText(ctx context.Context, filePath string, contentType string) (string, error)
}

type Processor struct{}

func NewProcessor() *Processor {
	return &Processor{}
}

func (p *Processor) ExtractText(ctx context.Context, filePath string, contentType string) (string, error) {
	switch contentType {
	case "application/pdf":
		return p.extractPDFText(filePath)
	case "text/plain":
		return p.extractPlainText(filePath)
	default:
		return "", fmt.Errorf("unsupported content type: %s", contentType)
	}
}

func (p *Processor) extractPDFText(filePath string) (string, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open PDF: %w", err)
	}
	defer f.Close()

	var buf bytes.Buffer
	totalPages := r.NumPage()

	for pageNum := 1; pageNum <= totalPages; pageNum++ {
		page := r.Page(pageNum)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}

		buf.WriteString(text)
		buf.WriteString("\n\n")
	}

	result := strings.TrimSpace(buf.String())
	if result == "" {
		return "", fmt.Errorf("no text could be extracted from PDF (may be scanned/image-based)")
	}

	return result, nil
}

func (p *Processor) extractPlainText(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(content), nil
}
