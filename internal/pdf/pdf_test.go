// Package pdf provides a pdfcpu-based PDF text extractor implementing port.PDFParser.
package pdf

import (
	"testing"

	"github.com/Tavo5691/job-searcher/internal/port"
)

// Compile-time check that PdfcpuParser implements port.PDFParser.
var _ port.PDFParser = (*PdfcpuParser)(nil)

func TestNewPdfcpuParser(t *testing.T) {
	p := NewPdfcpuParser()
	if p == nil {
		t.Error("NewPdfcpuParser must return a non-nil parser")
	}
}
