package main

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/sheets/v4"
)

const (
	maxSheetCells = 500000 // max total cells before rejecting (rows * cols)
	maxSheetTSV   = 5 << 20 // 5MB max TSV output size
)

// readSpreadsheet reads values from a Google Sheets spreadsheet and returns
// them formatted as TSV (tab-separated values).
//
// If rangeStr is empty, it fetches the first sheet's title from the spreadsheet
// metadata and reads all data from that sheet.
func readSpreadsheet(ctx context.Context, svc *sheets.Service, fileID string, rangeStr string) (string, error) {
	if rangeStr == "" {
		// Get spreadsheet metadata to find the first sheet name.
		if err := apiLimiter.Wait(ctx); err != nil {
			return "", fmt.Errorf("rate limited: %w", err)
		}
		metaCtx, metaCancel := withTimeout(ctx)
		defer metaCancel()

		spreadsheet, err := svc.Spreadsheets.Get(fileID).
			Context(metaCtx).
			Fields("sheets.properties.title").
			Do()
		if err != nil {
			return "", wrapAPIError(err, "getting spreadsheet metadata")
		}
		if len(spreadsheet.Sheets) == 0 {
			return "", fmt.Errorf("spreadsheet has no sheets")
		}
		if spreadsheet.Sheets[0].Properties == nil {
			return "", fmt.Errorf("sheet metadata missing")
		}
		title := spreadsheet.Sheets[0].Properties.Title
		rangeStr = fmt.Sprintf("'%s'", strings.ReplaceAll(title, "'", "''"))
	}

	if err := apiLimiter.Wait(ctx); err != nil {
		return "", fmt.Errorf("rate limited: %w", err)
	}
	valCtx, valCancel := withTimeout(ctx)
	defer valCancel()

	valueRange, err := svc.Spreadsheets.Values.Get(fileID, rangeStr).
		Context(valCtx).
		Do()
	if err != nil {
		return "", wrapAPIError(err, "reading spreadsheet values")
	}

	// Check total cell count to prevent OOM on huge sheets.
	if len(valueRange.Values) > 0 {
		maxCols := 0
		for _, row := range valueRange.Values {
			if len(row) > maxCols {
				maxCols = len(row)
			}
		}
		if len(valueRange.Values)*maxCols > maxSheetCells {
			return "", fmt.Errorf("spreadsheet range too large (%d rows x %d cols = %d cells, max %d). Use a specific range to limit data",
				len(valueRange.Values), maxCols, len(valueRange.Values)*maxCols, maxSheetCells)
		}
	}

	return formatValuesAsTSV(valueRange.Values), nil
}

// sanitizeTSVCell escapes characters that would corrupt TSV structure
// and prefixes formula-injection characters with a single quote.
func sanitizeTSVCell(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\t", "\\t")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	if len(s) > 0 && (s[0] == '=' || s[0] == '+' || s[0] == '-' || s[0] == '@') {
		s = "'" + s
	}
	return s
}

// formatValuesAsTSV converts a 2D slice of interface{} values into a
// tab-separated string. Each cell is sanitized to escape embedded tabs,
// newlines, and formula-injection characters. Rows are separated by newlines.
// Output is truncated at maxSheetTSV bytes.
func formatValuesAsTSV(values [][]interface{}) string {
	if len(values) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, row := range values {
		for j, cell := range row {
			if j > 0 {
				sb.WriteByte('\t')
			}
			var s string
			if cell != nil {
				s = fmt.Sprintf("%v", cell)
			}
			sb.WriteString(sanitizeTSVCell(s))
		}
		if i < len(values)-1 {
			sb.WriteByte('\n')
		}
		if sb.Len() > maxSheetTSV {
			sb.WriteString("\n\n[Sheet output truncated — exceeded 5MB text limit]")
			break
		}
	}
	return sb.String()
}
