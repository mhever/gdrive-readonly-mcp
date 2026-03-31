package main

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestRegisterToolsNoPanic(t *testing.T) {
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "gdrive-readonly-test",
			Version: "test",
		},
		nil,
	)
	// Should not panic.
	registerTools(server)
}

func TestTextResult(t *testing.T) {
	result := textResult("hello world")
	if result == nil {
		t.Fatal("textResult returned nil")
	}
	if result.IsError {
		t.Error("textResult should not be an error")
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(result.Content))
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	if tc.Text != "hello world" {
		t.Errorf("text = %q, want %q", tc.Text, "hello world")
	}
}

func TestErrorResult(t *testing.T) {
	result := errorResult(fmt.Errorf("something went wrong"))
	if result == nil {
		t.Fatal("errorResult returned nil")
	}
	if !result.IsError {
		t.Error("errorResult should have IsError = true")
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(result.Content))
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	if !strings.Contains(tc.Text, "something went wrong") {
		t.Errorf("error text = %q, want it to contain 'something went wrong'", tc.Text)
	}
}

func TestHandleSearchMissingQuery(t *testing.T) {
	result, _, err := handleSearch(context.Background(), &mcp.CallToolRequest{}, searchInput{Query: ""})
	if err != nil {
		t.Fatalf("handleSearch returned protocol error: %v", err)
	}
	if result == nil {
		t.Fatal("handleSearch returned nil result")
	}
	if !result.IsError {
		t.Error("expected IsError for empty query")
	}
}

func TestHandleGetFileMetadataInvalidID(t *testing.T) {
	tests := []struct {
		name   string
		fileID string
	}{
		{"empty", ""},
		{"injection", "abc' or 1=1 --"},
		{"too long", strings.Repeat("x", 201)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, err := handleGetFileMetadata(
				context.Background(),
				&mcp.CallToolRequest{},
				getFileMetadataInput{FileID: tt.fileID},
			)
			if err != nil {
				t.Fatalf("handleGetFileMetadata returned protocol error: %v", err)
			}
			if result == nil {
				t.Fatal("handleGetFileMetadata returned nil result")
			}
			if !result.IsError {
				t.Error("expected IsError for invalid file ID")
			}
		})
	}
}

func TestHandleListFilesPageTokenTooLong(t *testing.T) {
	result, _, err := handleListFiles(
		context.Background(),
		&mcp.CallToolRequest{},
		listFilesInput{PageToken: strings.Repeat("x", 501)},
	)
	if err != nil {
		t.Fatalf("handleListFiles returned protocol error: %v", err)
	}
	if result == nil {
		t.Fatal("handleListFiles returned nil result")
	}
	if !result.IsError {
		t.Error("expected IsError for too-long page token")
	}
}

func TestHandleSearchPageTokenTooLong(t *testing.T) {
	result, _, err := handleSearch(
		context.Background(),
		&mcp.CallToolRequest{},
		searchInput{Query: "test", PageToken: strings.Repeat("x", 501)},
	)
	if err != nil {
		t.Fatalf("handleSearch returned protocol error: %v", err)
	}
	if result == nil {
		t.Fatal("handleSearch returned nil result")
	}
	if !result.IsError {
		t.Error("expected IsError for too-long page token")
	}
}

func TestHandleListFilesInvalidFolderID(t *testing.T) {
	result, _, err := handleListFiles(
		context.Background(),
		&mcp.CallToolRequest{},
		listFilesInput{FolderID: "abc' DROP TABLE files --"},
	)
	if err != nil {
		t.Fatalf("handleListFiles returned protocol error: %v", err)
	}
	if result == nil {
		t.Fatal("handleListFiles returned nil result")
	}
	if !result.IsError {
		t.Error("expected IsError for invalid folder ID")
	}
}
