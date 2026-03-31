package main

import (
	"strings"
	"testing"

	"google.golang.org/api/docs/v1"
)

// helper to build a simple paragraph with one text run.
func textParagraph(text string) *docs.StructuralElement {
	return &docs.StructuralElement{
		Paragraph: &docs.Paragraph{
			Elements: []*docs.ParagraphElement{
				{TextRun: &docs.TextRun{Content: text}},
			},
		},
	}
}

func TestExtractStructuralElements_SimpleParagraph(t *testing.T) {
	elements := []*docs.StructuralElement{
		textParagraph("Hello, world!\n"),
	}
	var sb strings.Builder
	extractStructuralElements(&sb, elements)
	got := sb.String()
	if got != "Hello, world!\n" {
		t.Errorf("got %q, want %q", got, "Hello, world!\n")
	}
}

func TestExtractStructuralElements_MultipleParagraphs(t *testing.T) {
	elements := []*docs.StructuralElement{
		textParagraph("First paragraph.\n"),
		textParagraph("Second paragraph.\n"),
		textParagraph("Third paragraph.\n"),
	}
	var sb strings.Builder
	extractStructuralElements(&sb, elements)
	got := sb.String()
	want := "First paragraph.\nSecond paragraph.\nThird paragraph.\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExtractStructuralElements_MultipleTextRuns(t *testing.T) {
	// Simulates bold + normal mixed text within one paragraph.
	elements := []*docs.StructuralElement{
		{
			Paragraph: &docs.Paragraph{
				Elements: []*docs.ParagraphElement{
					{TextRun: &docs.TextRun{Content: "Bold text"}},
					{TextRun: &docs.TextRun{Content: " and normal text.\n"}},
				},
			},
		},
	}
	var sb strings.Builder
	extractStructuralElements(&sb, elements)
	got := sb.String()
	want := "Bold text and normal text.\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExtractStructuralElements_Table(t *testing.T) {
	elements := []*docs.StructuralElement{
		{
			Table: &docs.Table{
				TableRows: []*docs.TableRow{
					{
						TableCells: []*docs.TableCell{
							{Content: []*docs.StructuralElement{textParagraph("A1\n")}},
							{Content: []*docs.StructuralElement{textParagraph("B1\n")}},
						},
					},
					{
						TableCells: []*docs.TableCell{
							{Content: []*docs.StructuralElement{textParagraph("A2\n")}},
							{Content: []*docs.StructuralElement{textParagraph("B2\n")}},
						},
					},
				},
			},
		},
	}
	var sb strings.Builder
	extractStructuralElements(&sb, elements)
	got := sb.String()
	// Table cells are walked sequentially: A1, B1, A2, B2.
	if !strings.Contains(got, "A1") || !strings.Contains(got, "B1") ||
		!strings.Contains(got, "A2") || !strings.Contains(got, "B2") {
		t.Errorf("table extraction missing cells, got %q", got)
	}
}

func TestExtractStructuralElements_NestedTableInCell(t *testing.T) {
	// A table cell containing another table (nested).
	innerTable := &docs.StructuralElement{
		Table: &docs.Table{
			TableRows: []*docs.TableRow{
				{
					TableCells: []*docs.TableCell{
						{Content: []*docs.StructuralElement{textParagraph("nested cell\n")}},
					},
				},
			},
		},
	}
	elements := []*docs.StructuralElement{
		{
			Table: &docs.Table{
				TableRows: []*docs.TableRow{
					{
						TableCells: []*docs.TableCell{
							{Content: []*docs.StructuralElement{
								textParagraph("outer cell\n"),
								innerTable,
							}},
						},
					},
				},
			},
		},
	}
	var sb strings.Builder
	extractStructuralElements(&sb, elements)
	got := sb.String()
	if !strings.Contains(got, "outer cell") {
		t.Error("missing outer cell text")
	}
	if !strings.Contains(got, "nested cell") {
		t.Error("missing nested cell text")
	}
}

func TestExtractStructuralElements_EmptyDocument(t *testing.T) {
	var sb strings.Builder
	extractStructuralElements(&sb, nil)
	if sb.Len() != 0 {
		t.Errorf("expected empty string for nil elements, got %q", sb.String())
	}

	sb.Reset()
	extractStructuralElements(&sb, []*docs.StructuralElement{})
	if sb.Len() != 0 {
		t.Errorf("expected empty string for empty elements, got %q", sb.String())
	}
}

func TestExtractStructuralElements_WhitespaceOnly(t *testing.T) {
	elements := []*docs.StructuralElement{
		textParagraph("   \n"),
		textParagraph("\t\n"),
	}
	var sb strings.Builder
	extractStructuralElements(&sb, elements)
	got := sb.String()
	want := "   \n\t\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExtractStructuralElements_SectionBreak(t *testing.T) {
	elements := []*docs.StructuralElement{
		textParagraph("Before break.\n"),
		{SectionBreak: &docs.SectionBreak{}},
		textParagraph("After break.\n"),
	}
	var sb strings.Builder
	extractStructuralElements(&sb, elements)
	got := sb.String()
	want := "Before break.\n\nAfter break.\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExtractStructuralElements_TableOfContents(t *testing.T) {
	elements := []*docs.StructuralElement{
		{
			TableOfContents: &docs.TableOfContents{
				Content: []*docs.StructuralElement{
					textParagraph("Chapter 1\n"),
					textParagraph("Chapter 2\n"),
				},
			},
		},
	}
	var sb strings.Builder
	extractStructuralElements(&sb, elements)
	got := sb.String()
	if !strings.Contains(got, "Chapter 1") || !strings.Contains(got, "Chapter 2") {
		t.Errorf("table of contents extraction failed, got %q", got)
	}
}

func TestExtractStructuralElements_MixedContent(t *testing.T) {
	elements := []*docs.StructuralElement{
		textParagraph("Introduction\n"),
		{SectionBreak: &docs.SectionBreak{}},
		{
			Table: &docs.Table{
				TableRows: []*docs.TableRow{
					{
						TableCells: []*docs.TableCell{
							{Content: []*docs.StructuralElement{textParagraph("Header\n")}},
							{Content: []*docs.StructuralElement{textParagraph("Value\n")}},
						},
					},
				},
			},
		},
		textParagraph("Conclusion\n"),
	}
	var sb strings.Builder
	extractStructuralElements(&sb, elements)
	got := sb.String()
	if !strings.Contains(got, "Introduction") {
		t.Error("missing Introduction")
	}
	if !strings.Contains(got, "Header") {
		t.Error("missing Header")
	}
	if !strings.Contains(got, "Value") {
		t.Error("missing Value")
	}
	if !strings.Contains(got, "Conclusion") {
		t.Error("missing Conclusion")
	}
}

func TestExtractParagraph_NilParagraph(t *testing.T) {
	var sb strings.Builder
	extractParagraph(&sb, nil)
	if sb.Len() != 0 {
		t.Errorf("expected empty for nil paragraph, got %q", sb.String())
	}
}

func TestExtractParagraph_ElementWithNoTextRun(t *testing.T) {
	// ParagraphElement with InlineObjectElement (no TextRun) should be skipped.
	para := &docs.Paragraph{
		Elements: []*docs.ParagraphElement{
			{InlineObjectElement: &docs.InlineObjectElement{InlineObjectId: "obj1"}},
			{TextRun: &docs.TextRun{Content: "visible text"}},
		},
	}
	var sb strings.Builder
	extractParagraph(&sb, para)
	got := sb.String()
	if got != "visible text" {
		t.Errorf("got %q, want %q", got, "visible text")
	}
}

func TestExtractTable_NilTable(t *testing.T) {
	var sb strings.Builder
	extractTable(&sb, nil)
	if sb.Len() != 0 {
		t.Errorf("expected empty for nil table, got %q", sb.String())
	}
}
