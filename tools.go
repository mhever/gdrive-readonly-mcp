package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// textResult returns a successful CallToolResult with text content.
func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}
}

// errorResult returns a CallToolResult marked as an error.
func errorResult(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: err.Error()},
		},
	}
}

// --- Tool input types ---

type listFilesInput struct {
	Query     string `json:"query,omitempty" jsonschema:"filter files by name"`
	FolderID  string `json:"folder_id,omitempty" jsonschema:"list files in this folder ID"`
	PageSize  int    `json:"page_size,omitempty" jsonschema:"number of results per page (default 20, max 100)"`
	PageToken string `json:"page_token,omitempty" jsonschema:"token for the next page of results"`
}

type searchInput struct {
	Query     string `json:"query" jsonschema:"search query to find files by name or content"`
	PageSize  int    `json:"page_size,omitempty" jsonschema:"number of results per page (default 20, max 100)"`
	PageToken string `json:"page_token,omitempty" jsonschema:"token for the next page of results"`
}

type getFileMetadataInput struct {
	FileID string `json:"file_id" jsonschema:"Google Drive file ID"`
}

// --- Tool handlers ---

func handleListFiles(ctx context.Context, req *mcp.CallToolRequest, input listFilesInput) (*mcp.CallToolResult, any, error) {
	if input.FolderID != "" {
		if err := validateFileID(input.FolderID); err != nil {
			return errorResult(fmt.Errorf("invalid folder_id: %w", err)), nil, nil
		}
	}
	if len(input.PageToken) > 500 {
		return errorResult(fmt.Errorf("page_token too long (%d chars, max 500)", len(input.PageToken))), nil, nil
	}

	result, err := listFiles(ctx, driveSvc, input.Query, input.FolderID, input.PageSize, input.PageToken)
	if err != nil {
		return errorResult(err), nil, nil
	}

	return textResult(formatFileList(result)), nil, nil
}

func handleSearch(ctx context.Context, req *mcp.CallToolRequest, input searchInput) (*mcp.CallToolResult, any, error) {
	if input.Query == "" {
		return errorResult(fmt.Errorf("query is required")), nil, nil
	}
	if len(input.PageToken) > 500 {
		return errorResult(fmt.Errorf("page_token too long (%d chars, max 500)", len(input.PageToken))), nil, nil
	}

	result, err := searchFiles(ctx, driveSvc, input.Query, input.PageSize, input.PageToken)
	if err != nil {
		return errorResult(err), nil, nil
	}

	return textResult(formatFileList(result)), nil, nil
}

func handleGetFileMetadata(ctx context.Context, req *mcp.CallToolRequest, input getFileMetadataInput) (*mcp.CallToolResult, any, error) {
	if input.FileID == "" {
		return errorResult(fmt.Errorf("file_id is required")), nil, nil
	}
	if err := validateFileID(input.FileID); err != nil {
		return errorResult(fmt.Errorf("invalid file_id: %w", err)), nil, nil
	}

	file, err := getFileMetadata(ctx, driveSvc, input.FileID)
	if err != nil {
		return errorResult(err), nil, nil
	}

	// Marshal to indented JSON for readability.
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return errorResult(fmt.Errorf("failed to format metadata: %w", err)), nil, nil
	}

	return textResult(string(data)), nil, nil
}

func registerTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "gdrive_list_files",
		Description: "List files in Google Drive, optionally filtered by name query and/or folder ID. Returns file names, IDs, types, and modification times.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}, handleListFiles)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "gdrive_search",
		Description: "Search for files in Google Drive by name or content. Returns matching files with their IDs, types, and modification times.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}, handleSearch)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "gdrive_get_file_metadata",
		Description: "Get detailed metadata for a specific Google Drive file by its ID. Returns all available file properties.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}, handleGetFileMetadata)
}
