package main

import (
	"context"
	"strings"

	"google.golang.org/api/docs/v1"
)

const maxDocSize = 5 << 20 // 5MB max extracted text

// readDocument fetches a Google Doc and extracts its plain text content.
// It walks the document body's structural elements to extract text from
// paragraphs, tables, section breaks, and table of contents entries.
func readDocument(ctx context.Context, svc *docs.Service, fileID string) (string, error) {
	doc, err := svc.Documents.Get(fileID).Context(ctx).Do()
	if err != nil {
		return "", wrapAPIError(err, "reading document")
	}

	if doc.Body == nil {
		return "", nil
	}

	var sb strings.Builder
	extractStructuralElements(&sb, doc.Body.Content)
	if sb.Len() > maxDocSize {
		s := sb.String()[:maxDocSize]
		return s + "\n\n[Document truncated — exceeded 5MB text limit]", nil
	}
	return sb.String(), nil
}

// extractStructuralElements walks a slice of StructuralElements and appends
// their plain text content to the builder.
func extractStructuralElements(sb *strings.Builder, elements []*docs.StructuralElement) {
	for _, elem := range elements {
		switch {
		case elem.Paragraph != nil:
			extractParagraph(sb, elem.Paragraph)
		case elem.Table != nil:
			extractTable(sb, elem.Table)
		case elem.SectionBreak != nil:
			sb.WriteString("\n")
		case elem.TableOfContents != nil:
			// TableOfContents has a Content field with structural elements.
			extractStructuralElements(sb, elem.TableOfContents.Content)
		}
	}
}

// extractParagraph extracts text from a paragraph's elements.
func extractParagraph(sb *strings.Builder, para *docs.Paragraph) {
	if para == nil {
		return
	}
	for _, elem := range para.Elements {
		if elem.TextRun != nil {
			sb.WriteString(elem.TextRun.Content)
		}
	}
}

// extractTable walks all rows and cells in a table, extracting text content.
func extractTable(sb *strings.Builder, table *docs.Table) {
	if table == nil {
		return
	}
	for _, row := range table.TableRows {
		for _, cell := range row.TableCells {
			// Each cell contains structural elements (paragraphs, nested tables, etc.)
			extractStructuralElements(sb, cell.Content)
		}
	}
}
