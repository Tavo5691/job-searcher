// Package pdf implements the port.PDFParser interface using pdfcpu.
package pdf

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// PdfcpuParser implements port.PDFParser using the pdfcpu library.
type PdfcpuParser struct{}

// NewPdfcpuParser returns a new PdfcpuParser.
func NewPdfcpuParser() *PdfcpuParser {
	return &PdfcpuParser{}
}

// ExtractText reads the PDF at path and returns its text content.
// It uses pdfcpu to extract page content streams, which are then joined.
func (p *PdfcpuParser) ExtractText(_ context.Context, path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("pdf extract: open %s: %w", path, err)
	}
	defer f.Close()

	conf := model.NewDefaultConfiguration()
	ctx, err := pdfapi.ReadValidateAndOptimize(f, conf)
	if err != nil {
		return "", fmt.Errorf("pdf extract: validate %s: %w", path, err)
	}

	var sb strings.Builder
	baseName := strings.TrimSuffix(filepath.Base(path), ".pdf")
	tmpDir, err := os.MkdirTemp("", "pdfcpu-*")
	if err != nil {
		return "", fmt.Errorf("pdf extract: create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Seek back to beginning so ExtractContent can re-read.
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("pdf extract: seek: %w", err)
	}

	if err := pdfapi.ExtractContent(f, tmpDir, baseName, nil, conf); err != nil {
		return "", fmt.Errorf("pdf extract: extract content: %w", err)
	}

	// Read all generated content files.
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return "", fmt.Errorf("pdf extract: read dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(tmpDir, entry.Name()))
		if err != nil {
			return "", fmt.Errorf("pdf extract: read content file %s: %w", entry.Name(), err)
		}
		sb.Write(stripPDFOperators(data))
		sb.WriteByte('\n')
	}

	_ = ctx // ctx was used implicitly during extraction
	return sb.String(), nil
}

// stripPDFOperators does a best-effort extraction of text from PDF content streams.
// It pulls out text between Tj/TJ operators.
func stripPDFOperators(data []byte) []byte {
	var out bytes.Buffer
	// Simple heuristic: extract parenthesised strings (PDF text objects).
	inStr := false
	for _, b := range data {
		if b == '(' && !inStr {
			inStr = true
			continue
		}
		if b == ')' && inStr {
			inStr = false
			out.WriteByte(' ')
			continue
		}
		if inStr {
			if b >= 0x20 && b < 0x7F {
				out.WriteByte(b)
			}
		}
	}
	return out.Bytes()
}
