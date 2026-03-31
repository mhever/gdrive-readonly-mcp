package main

import "strings"

// escapeQuery escapes user input for use in Google Drive API query strings.
// It escapes backslashes and single quotes, which are the special characters
// in Drive query string literals.
func escapeQuery(input string) string {
	// Escape backslashes first (order matters), then single quotes.
	s := strings.ReplaceAll(input, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	return s
}
