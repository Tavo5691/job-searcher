package port

import "context"

// PDFParser extracts plain text from a PDF file.
// Implementations: internal/pdf.PdfcpuParser.
type PDFParser interface {
	ExtractText(ctx context.Context, path string) (string, error)
}
