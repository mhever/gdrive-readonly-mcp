package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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

type readFileInput struct {
	FileID string `json:"file_id" jsonschema:"Google Drive file ID to read"`
}

type readSheetInput struct {
	FileID string `json:"file_id" jsonschema:"Google Sheets spreadsheet ID"`
	Range  string `json:"range,omitempty" jsonschema:"optional A1 notation range (e.g. Sheet1!A1:E10); if omitted reads all data from first sheet"`
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

func handleReadFile(ctx context.Context, req *mcp.CallToolRequest, input readFileInput) (*mcp.CallToolResult, any, error) {
	if input.FileID == "" {
		return errorResult(fmt.Errorf("file_id is required")), nil, nil
	}
	if err := validateFileID(input.FileID); err != nil {
		return errorResult(fmt.Errorf("invalid file_id: %w", err)), nil, nil
	}

	meta, err := getFileMetadata(ctx, driveSvc, input.FileID)
	if err != nil {
		return errorResult(err), nil, nil
	}

	switch meta.MimeType {
	case "application/vnd.google-apps.document":
		text, err := readDocument(ctx, docsSvc, input.FileID)
		if err != nil {
			return errorResult(err), nil, nil
		}
		return textResult(text), nil, nil

	case "application/vnd.google-apps.spreadsheet":
		text, err := readSpreadsheet(ctx, sheetsSvc, input.FileID, "")
		if err != nil {
			return errorResult(err), nil, nil
		}
		return textResult(text), nil, nil

	default:
		// Check for other unsupported Google Apps types.
		if strings.HasPrefix(meta.MimeType, "application/vnd.google-apps.") {
			return errorResult(fmt.Errorf("Unsupported Google Apps type: %s. This server supports Docs and Sheets.", meta.MimeType)), nil, nil
		}

		// Regular file — check MIME type before downloading.
		if !isTextMime(meta.MimeType) {
			return errorResult(fmt.Errorf("Binary file (%s) cannot be displayed as text", meta.MimeType)), nil, nil
		}
		data, _, err := downloadFile(ctx, driveSvc, input.FileID)
		if err != nil {
			return errorResult(err), nil, nil
		}
		return textResult(string(data)), nil, nil
	}
}

func handleReadSheet(ctx context.Context, req *mcp.CallToolRequest, input readSheetInput) (*mcp.CallToolResult, any, error) {
	if input.FileID == "" {
		return errorResult(fmt.Errorf("file_id is required")), nil, nil
	}
	if err := validateFileID(input.FileID); err != nil {
		return errorResult(fmt.Errorf("invalid file_id: %w", err)), nil, nil
	}

	if len(input.Range) > 500 {
		return errorResult(fmt.Errorf("range too long (%d chars, max 500)", len(input.Range))), nil, nil
	}

	text, err := readSpreadsheet(ctx, sheetsSvc, input.FileID, input.Range)
	if err != nil {
		return errorResult(err), nil, nil
	}
	return textResult(text), nil, nil
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

	mcp.AddTool(server, &mcp.Tool{
		Name:        "gdrive_read_file",
		Description: "Read the content of a Google Drive file. Supports Google Docs (text extraction), Google Sheets (first sheet as TSV), and regular text files (direct download). Binary files and unsupported Google Apps types return an error.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}, handleReadFile)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "gdrive_read_sheet",
		Description: "Read values from a Google Sheets spreadsheet, optionally specifying an A1 notation range. Returns data formatted as TSV. If no range is specified, reads all data from the first sheet.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}, handleReadSheet)
}
