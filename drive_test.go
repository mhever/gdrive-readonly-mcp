package main

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
)

func TestValidateFileID(t *testing.T) {
	tests := []struct {
		name    string
		fileID  string
		wantErr bool
	}{
		{name: "valid simple", fileID: "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgVE2upms", wantErr: false},
		{name: "valid with hyphens", fileID: "abc-def-123", wantErr: false},
		{name: "valid with underscores", fileID: "abc_def_123", wantErr: false},
		{name: "valid single char", fileID: "a", wantErr: false},
		{name: "empty string", fileID: "", wantErr: true},
		{name: "too long", fileID: strings.Repeat("a", 201), wantErr: true},
		{name: "max length", fileID: strings.Repeat("a", 200), wantErr: false},
		{name: "contains space", fileID: "abc def", wantErr: true},
		{name: "contains dot", fileID: "abc.def", wantErr: true},
		{name: "contains slash", fileID: "abc/def", wantErr: true},
		{name: "contains single quote", fileID: "abc'def", wantErr: true},
		{name: "injection attempt with quote", fileID: "abc' or 1=1 --", wantErr: true},
		{name: "injection attempt with backslash", fileID: `abc\def`, wantErr: true},
		{name: "unicode characters", fileID: "abc文件", wantErr: true},
		{name: "contains colon", fileID: "abc:def", wantErr: true},
		{name: "contains at sign", fileID: "abc@def", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFileID(tt.fileID)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFileID(%q) error = %v, wantErr %v", tt.fileID, err, tt.wantErr)
			}
		})
	}
}

func TestIsTextMime(t *testing.T) {
	tests := []struct {
		mimeType string
		want     bool
	}{
		{"text/plain", true},
		{"text/html", true},
		{"text/csv", true},
		{"text/xml", true},
		{"application/json", true},
		{"application/xml", true},
		{"application/javascript", true},
		{"application/atom+xml", true},
		{"application/rss+xml", true},
		{"application/vnd.api+json", true},
		{"application/ld+json", true},
		{"text/markdown", true},
		{"text/x-markdown", true},
		{"application/yaml", true},
		{"application/x-yaml", true},
		{"application/toml", true},
		{"application/x-sh", true},
		{"application/x-shellscript", true},
		{"application/sql", true},
		{"application/typescript", true},
		{"application/x-typescript", true},
		{"application/octet-stream", false},
		{"image/png", false},
		{"image/jpeg", false},
		{"application/pdf", false},
		{"application/zip", false},
		{"video/mp4", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			got := isTextMime(tt.mimeType)
			if got != tt.want {
				t.Errorf("isTextMime(%q) = %v, want %v", tt.mimeType, got, tt.want)
			}
		})
	}
}

func TestFormatFileList(t *testing.T) {
	t.Run("nil file list", func(t *testing.T) {
		got := formatFileList(nil)
		if got != "No files found." {
			t.Errorf("formatFileList(nil) = %q, want 'No files found.'", got)
		}
	})

	t.Run("empty file list", func(t *testing.T) {
		got := formatFileList(&drive.FileList{})
		if got != "No files found." {
			t.Errorf("formatFileList(empty) = %q, want 'No files found.'", got)
		}
	})

	t.Run("with files", func(t *testing.T) {
		fl := &drive.FileList{
			Files: []*drive.File{
				{
					Id:           "abc123",
					Name:         "Test Document",
					MimeType:     "application/vnd.google-apps.document",
					ModifiedTime: "2026-01-15T10:30:00Z",
					Size:         0, // Google Apps files have no size
				},
				{
					Id:           "def456",
					Name:         "photo.jpg",
					MimeType:     "image/jpeg",
					ModifiedTime: "2026-02-20T14:00:00Z",
					Size:         1048576,
				},
			},
		}
		got := formatFileList(fl)
		if !strings.Contains(got, "Showing 2 file(s)") {
			t.Error("missing file count")
		}
		if !strings.Contains(got, "Test Document") {
			t.Error("missing file name 'Test Document'")
		}
		if !strings.Contains(got, "abc123") {
			t.Error("missing file ID 'abc123'")
		}
		if !strings.Contains(got, "photo.jpg") {
			t.Error("missing file name 'photo.jpg'")
		}
		if !strings.Contains(got, "1048576 bytes") {
			t.Error("missing size for photo.jpg")
		}
		// Google Apps file should NOT show size
		if strings.Contains(got, "Size: 0 bytes") {
			t.Error("should not show size 0 for Google Apps files")
		}
	})

	t.Run("with next page token", func(t *testing.T) {
		fl := &drive.FileList{
			Files: []*drive.File{
				{Id: "a", Name: "file.txt", MimeType: "text/plain"},
			},
			NextPageToken: "token123",
		}
		got := formatFileList(fl)
		if !strings.Contains(got, "Next page token: token123") {
			t.Error("missing next page token")
		}
		if !strings.Contains(got, "more results available") {
			t.Error("missing more results hint")
		}
	})

	t.Run("without next page token", func(t *testing.T) {
		fl := &drive.FileList{
			Files: []*drive.File{
				{Id: "a", Name: "file.txt", MimeType: "text/plain"},
			},
		}
		got := formatFileList(fl)
		if strings.Contains(got, "Next page token") {
			t.Error("should not contain next page token when empty")
		}
	})
}

func TestWrapAPIError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		got := wrapAPIError(nil, "test")
		if got != nil {
			t.Errorf("wrapAPIError(nil) = %v, want nil", got)
		}
	})

	t.Run("404 not found", func(t *testing.T) {
		err := &googleapi.Error{Code: 404, Message: "File not found"}
		got := wrapAPIError(err, "getting file")
		if !strings.Contains(got.Error(), "not found") {
			t.Errorf("expected 'not found' in error, got %q", got.Error())
		}
	})

	t.Run("403 access denied", func(t *testing.T) {
		err := &googleapi.Error{Code: 403, Message: "Insufficient permissions"}
		got := wrapAPIError(err, "getting file")
		if !strings.Contains(got.Error(), "access denied") {
			t.Errorf("expected 'access denied' in error, got %q", got.Error())
		}
	})

	t.Run("429 rate limited", func(t *testing.T) {
		err := &googleapi.Error{Code: 429, Message: "Too many requests"}
		got := wrapAPIError(err, "listing files")
		if !strings.Contains(got.Error(), "rate limited") {
			t.Errorf("expected 'rate limited' in error, got %q", got.Error())
		}
	})

	t.Run("other google API error", func(t *testing.T) {
		err := &googleapi.Error{Code: 500, Message: "Internal server error"}
		got := wrapAPIError(err, "operation")
		if !strings.Contains(got.Error(), "500") {
			t.Errorf("expected error code in message, got %q", got.Error())
		}
		if !strings.Contains(got.Error(), "Internal server error") {
			t.Errorf("expected error message in output, got %q", got.Error())
		}
	})

	t.Run("non-google error", func(t *testing.T) {
		err := fmt.Errorf("network timeout")
		got := wrapAPIError(err, "operation")
		if !strings.Contains(got.Error(), "network timeout") {
			t.Errorf("expected original error in output, got %q", got.Error())
		}
		if !strings.Contains(got.Error(), "operation") {
			t.Errorf("expected context in output, got %q", got.Error())
		}
	})
}

func TestQueryBuildingEdgeCases(t *testing.T) {
	// These test that escapeQuery is used correctly in query construction.
	// We can't call listFiles/searchFiles without a real service, but we
	// can verify the escaping behavior used in query building.

	t.Run("search query with single quotes", func(t *testing.T) {
		input := "it's a test"
		escaped := escapeQuery(input)
		q := fmt.Sprintf("name contains '%s' or fullText contains '%s'", escaped, escaped)
		if !strings.Contains(q, `it\'s a test`) {
			t.Errorf("query not properly escaped: %s", q)
		}
	})

	t.Run("search query with backslashes", func(t *testing.T) {
		input := `path\to\file`
		escaped := escapeQuery(input)
		q := fmt.Sprintf("name contains '%s'", escaped)
		if !strings.Contains(q, `path\\to\\file`) {
			t.Errorf("query not properly escaped: %s", q)
		}
	})

	t.Run("folder ID validation rejects special chars", func(t *testing.T) {
		// A folder ID with injection-style content should fail validation.
		err := validateFileID("abc' in parents or '1'='1")
		if err == nil {
			t.Error("expected error for injection attempt in folder ID")
		}
	})
}

func TestDownloadFileRejectsInvalidIDs(t *testing.T) {
	tests := []struct {
		name   string
		fileID string
	}{
		{"empty", ""},
		{"injection attempt", "abc' or 1=1 --"},
		{"too long", strings.Repeat("x", 201)},
		{"contains slash", "abc/def"},
		{"contains space", "abc def"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// downloadFile should reject invalid file IDs before making any API call.
			_, _, err := downloadFile(context.Background(), nil, tt.fileID)
			if err == nil {
				t.Errorf("downloadFile(%q) should have returned an error", tt.fileID)
			}
		})
	}
}

func TestFormatFileListTruncatesLongNames(t *testing.T) {
	longName := strings.Repeat("a", 600)
	fl := &drive.FileList{
		Files: []*drive.File{
			{Id: "abc", Name: longName, MimeType: "text/plain"},
		},
	}
	got := formatFileList(fl)
	if strings.Contains(got, longName) {
		t.Error("long file name should be truncated")
	}
	truncated := strings.Repeat("a", 500) + "..."
	if !strings.Contains(got, truncated) {
		t.Error("expected truncated name with '...' suffix")
	}
}
