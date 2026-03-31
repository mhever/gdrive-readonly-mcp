package main

import (
	"strings"
	"testing"
)

func TestSanitizeTSVCell(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "hello", "hello"},
		{"tab", "a\tb", "a\\tb"},
		{"newline", "a\nb", "a\\nb"},
		{"carriage return", "a\rb", "a\\rb"},
		{"backslash", "a\\b", "a\\\\b"},
		{"formula equals", "=SUM(A1)", "'=SUM(A1)"},
		{"formula plus", "+1", "'+1"},
		{"formula minus", "-1", "'-1"},
		{"formula at", "@import", "'@import"},
		{"backslash then formula", "\\=test", "\\\\=test"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeTSVCell(tt.in)
			if got != tt.want {
				t.Errorf("sanitizeTSVCell(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestFormatValuesAsTSV_NormalGrid(t *testing.T) {
	values := [][]interface{}{
		{"Name", "Age", "Active"},
		{"Alice", 30, true},
		{"Bob", 25, false},
	}
	got := formatValuesAsTSV(values)
	want := "Name\tAge\tActive\nAlice\t30\ttrue\nBob\t25\tfalse"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatValuesAsTSV_EmptySheet(t *testing.T) {
	got := formatValuesAsTSV(nil)
	if got != "" {
		t.Errorf("expected empty string for nil values, got %q", got)
	}

	got = formatValuesAsTSV([][]interface{}{})
	if got != "" {
		t.Errorf("expected empty string for empty values, got %q", got)
	}
}

func TestFormatValuesAsTSV_SingleCell(t *testing.T) {
	values := [][]interface{}{
		{"hello"},
	}
	got := formatValuesAsTSV(values)
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestFormatValuesAsTSV_SpecialCharacters(t *testing.T) {
	// Cells with tabs and newlines embedded in them should be escaped.
	values := [][]interface{}{
		{"has\ttab", "has\nnewline"},
		{"normal", "also\tnormal"},
	}
	got := formatValuesAsTSV(values)
	// Tabs and newlines in cell values should be escaped.
	if !strings.Contains(got, "has\\ttab") {
		t.Errorf("expected escaped tab in output, got %q", got)
	}
	if !strings.Contains(got, "has\\nnewline") {
		t.Errorf("expected escaped newline in output, got %q", got)
	}
	if !strings.Contains(got, "also\\tnormal") {
		t.Errorf("expected escaped tab in second row, got %q", got)
	}
}

func TestFormatValuesAsTSV_FormulaInjection(t *testing.T) {
	values := [][]interface{}{
		{"=SUM(A1:A2)", "+cmd", "-exec", "@import"},
		{"safe", "also safe", "123", "hello"},
	}
	got := formatValuesAsTSV(values)
	lines := strings.Split(got, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), got)
	}
	cols := strings.Split(lines[0], "\t")
	if len(cols) != 4 {
		t.Fatalf("expected 4 columns, got %d: %v", len(cols), cols)
	}
	if cols[0] != "'=SUM(A1:A2)" {
		t.Errorf("expected formula prefix for =, got %q", cols[0])
	}
	if cols[1] != "'+cmd" {
		t.Errorf("expected formula prefix for +, got %q", cols[1])
	}
	if cols[2] != "'-exec" {
		t.Errorf("expected formula prefix for -, got %q", cols[2])
	}
	if cols[3] != "'@import" {
		t.Errorf("expected formula prefix for @, got %q", cols[3])
	}
}

func TestFormatValuesAsTSV_RaggedRows(t *testing.T) {
	// Rows with different numbers of columns.
	values := [][]interface{}{
		{"A", "B", "C"},
		{"D"},
		{"E", "F"},
	}
	got := formatValuesAsTSV(values)
	lines := strings.Split(got, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), got)
	}
	if lines[0] != "A\tB\tC" {
		t.Errorf("line 0: got %q, want %q", lines[0], "A\tB\tC")
	}
	if lines[1] != "D" {
		t.Errorf("line 1: got %q, want %q", lines[1], "D")
	}
	if lines[2] != "E\tF" {
		t.Errorf("line 2: got %q, want %q", lines[2], "E\tF")
	}
}

func TestFormatValuesAsTSV_EmptyRow(t *testing.T) {
	values := [][]interface{}{
		{"header1", "header2"},
		{},
		{"data1", "data2"},
	}
	got := formatValuesAsTSV(values)
	lines := strings.Split(got, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), got)
	}
	// Empty row should produce an empty line.
	if lines[1] != "" {
		t.Errorf("expected empty line for empty row, got %q", lines[1])
	}
}

func TestFormatValuesAsTSV_MixedTypes(t *testing.T) {
	values := [][]interface{}{
		{"string", 42, 3.14, true, nil},
	}
	got := formatValuesAsTSV(values)
	want := "string\t42\t3.14\ttrue\t"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatValuesAsTSV_LargeDataset(t *testing.T) {
	// Verify formatting works correctly with a larger dataset.
	rows := 100
	cols := 10
	values := make([][]interface{}, rows)
	for i := range values {
		values[i] = make([]interface{}, cols)
		for j := range values[i] {
			values[i][j] = i*cols + j
		}
	}
	got := formatValuesAsTSV(values)
	lines := strings.Split(got, "\n")
	if len(lines) != rows {
		t.Fatalf("expected %d lines, got %d", rows, len(lines))
	}
	// Check first row.
	firstCols := strings.Split(lines[0], "\t")
	if len(firstCols) != cols {
		t.Errorf("expected %d columns in first row, got %d", cols, len(firstCols))
	}
	if firstCols[0] != "0" || firstCols[9] != "9" {
		t.Errorf("first row values unexpected: %v", firstCols)
	}
}
