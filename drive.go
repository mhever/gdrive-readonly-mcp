package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

const maxDownloadSize = 1 << 20 // 1MB

// Package-level services — initialized in main.go via initServices.
var (
	driveSvc  *drive.Service
	docsSvc   *docs.Service
	sheetsSvc *sheets.Service
)

// initServices creates Google API service clients from an authenticated HTTP client.
func initServices(ctx context.Context, client *http.Client) error {
	var err error
	driveSvc, err = drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("failed to create Drive service: %w", err)
	}
	docsSvc, err = docs.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("failed to create Docs service: %w", err)
	}
	sheetsSvc, err = sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("failed to create Sheets service: %w", err)
	}
	return nil
}

// validateFileID checks that a file ID has a valid format:
// alphanumeric, hyphens, and underscores only, 1-200 characters.
func validateFileID(fileID string) error {
	if fileID == "" {
		return fmt.Errorf("file ID is required")
	}
	if len(fileID) > 200 {
		return fmt.Errorf("file ID too long (%d chars, max 200)", len(fileID))
	}
	for _, r := range fileID {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return fmt.Errorf("file ID contains invalid character %q", r)
		}
	}
	return nil
}

// listFiles queries Drive for files, optionally within a folder.
// If folderID is set, it prepends a parent filter. User input in query is escaped.
// Default pageSize is 20, capped at 100.
func listFiles(ctx context.Context, svc *drive.Service, query, folderID string, pageSize int, pageToken string) (*drive.FileList, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	var parts []string
	if folderID != "" {
		if err := validateFileID(folderID); err != nil {
			return nil, fmt.Errorf("invalid folder ID: %w", err)
		}
		parts = append(parts, fmt.Sprintf("'%s' in parents", escapeQuery(folderID)))
	}
	if query != "" {
		parts = append(parts, fmt.Sprintf("(name contains '%s')", escapeQuery(query)))
	}

	q := strings.Join(parts, " and ")

	call := svc.Files.List().
		PageSize(int64(pageSize)).
		Fields("nextPageToken, files(id,name,mimeType,modifiedTime,size,parents)")

	if q != "" {
		call = call.Q(q)
	}
	if pageToken != "" {
		call = call.PageToken(pageToken)
	}

	if err := apiLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limited: %w", err)
	}
	callCtx, cancel := withTimeout(ctx)
	defer cancel()

	result, err := call.Context(callCtx).Do()
	if err != nil {
		return nil, wrapAPIError(err, "listing files")
	}
	return result, nil
}

// searchFiles searches Drive for files matching the query.
// It searches by filename first. If no results are found and pageToken is empty,
// it falls back to a full-text content search.
// Default pageSize is 20, capped at 100.
func searchFiles(ctx context.Context, svc *drive.Service, query string, pageSize int, pageToken string) (*drive.FileList, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	escaped := escapeQuery(query)
	fields := "nextPageToken, files(id,name,mimeType,modifiedTime,size)"

	// First: search by filename.
	nameQ := fmt.Sprintf("name contains '%s'", escaped)

	call := svc.Files.List().
		Q(nameQ).
		PageSize(int64(pageSize)).
		Fields(googleapi.Field(fields))

	if pageToken != "" {
		call = call.PageToken(pageToken)
	}

	if err := apiLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limited: %w", err)
	}
	callCtx, cancel := withTimeout(ctx)
	defer cancel()

	result, err := call.Context(callCtx).Do()
	if err != nil {
		return nil, wrapAPIError(err, "searching files by name")
	}

	// If name search found results (or we're paginating), return them.
	if len(result.Files) > 0 || pageToken != "" {
		return result, nil
	}

	// Fallback: search by full-text content.
	// Content-fallback results are not paginated to avoid pageToken
	// cross-contamination between the two different query types.
	contentQ := fmt.Sprintf("fullText contains '%s'", escaped)

	if err := apiLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limited: %w", err)
	}
	fallbackCtx, fallbackCancel := withTimeout(ctx)
	defer fallbackCancel()

	result, err = svc.Files.List().
		Q(contentQ).
		PageSize(int64(pageSize)).
		Fields(googleapi.Field(fields)).
		Context(fallbackCtx).Do()
	if err != nil {
		return nil, wrapAPIError(err, "searching files by content")
	}

	// Clear NextPageToken to prevent the caller from paginating
	// content results with a token that would be misrouted to the name query.
	result.NextPageToken = ""
	return result, nil
}

// getFileMetadata retrieves full metadata for a file.
func getFileMetadata(ctx context.Context, svc *drive.Service, fileID string) (*drive.File, error) {
	if err := validateFileID(fileID); err != nil {
		return nil, err
	}

	if err := apiLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limited: %w", err)
	}
	callCtx, cancel := withTimeout(ctx)
	defer cancel()

	file, err := svc.Files.Get(fileID).Context(callCtx).Fields("*").Do()
	if err != nil {
		return nil, wrapAPIError(err, "getting file metadata")
	}
	return file, nil
}

// downloadFile downloads a non-Google-Apps file, enforcing a size cap of maxDownloadSize.
// Returns the file bytes and its MIME type.
// If meta is non-nil, its Size and MimeType are used and the metadata API call is skipped.
// If meta is nil, metadata is fetched first.
func downloadFile(ctx context.Context, svc *drive.Service, fileID string, meta *drive.File) ([]byte, string, error) {
	if err := validateFileID(fileID); err != nil {
		return nil, "", err
	}

	if meta == nil {
		// Fetch metadata to check size before downloading.
		if err := apiLimiter.Wait(ctx); err != nil {
			return nil, "", fmt.Errorf("rate limited: %w", err)
		}
		metaCtx, metaCancel := withTimeout(ctx)
		defer metaCancel()

		var err error
		meta, err = svc.Files.Get(fileID).Context(metaCtx).Fields("size,mimeType").Do()
		if err != nil {
			return nil, "", wrapAPIError(err, "getting file metadata for download")
		}
	}

	// Note: The metadata size check above is defense-in-depth only.
	// The LimitReader below is the actual enforcement mechanism,
	// preventing more than maxDownloadSize bytes from being read
	// regardless of what the metadata reports.
	if meta.Size > int64(maxDownloadSize) {
		return nil, "", fmt.Errorf("file too large (%d bytes, max %d bytes)", meta.Size, maxDownloadSize)
	}

	if err := apiLimiter.Wait(ctx); err != nil {
		return nil, "", fmt.Errorf("rate limited: %w", err)
	}
	dlCtx, dlCancel := withTimeout(ctx)
	defer dlCancel()

	resp, err := svc.Files.Get(fileID).Context(dlCtx).Download()
	if err != nil {
		return nil, "", wrapAPIError(err, "downloading file")
	}
	defer resp.Body.Close()

	// Read up to maxDownloadSize + 1 to detect oversized responses even if
	// the metadata size field was inaccurate (e.g. for Google Apps types).
	data, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxDownloadSize)+1))
	if err != nil {
		return nil, "", fmt.Errorf("reading file content: %w", err)
	}
	if len(data) > maxDownloadSize {
		return nil, "", fmt.Errorf("file too large (exceeded %d byte limit during download)", maxDownloadSize)
	}

	return data, meta.MimeType, nil
}

// wrapAPIError provides human-readable messages for common Google API errors.
func wrapAPIError(err error, operation string) error {
	if err == nil {
		return nil
	}
	gErr, ok := err.(*googleapi.Error)
	if !ok {
		return fmt.Errorf("%s: %w", operation, err)
	}
	switch gErr.Code {
	case 404:
		return fmt.Errorf("%s: not found: %w", operation, err)
	case 403:
		return fmt.Errorf("%s: access denied: %w", operation, err)
	case 429:
		return fmt.Errorf("%s: rate limited — try again later: %w", operation, err)
	default:
		return fmt.Errorf("%s: API error %d — %s: %w", operation, gErr.Code, gErr.Message, err)
	}
}

// formatFileList formats a Drive FileList into human-readable text for LLM consumption.
func formatFileList(fileList *drive.FileList) string {
	if fileList == nil || len(fileList.Files) == 0 {
		return "No files found."
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Showing %d file(s):\n\n", len(fileList.Files))

	for _, f := range fileList.Files {
		name := f.Name
		if len(name) > 500 {
			name = name[:500] + "..."
		}
		fmt.Fprintf(&sb, "- %s\n", name)
		fmt.Fprintf(&sb, "  ID: %s\n", f.Id)
		fmt.Fprintf(&sb, "  Type: %s\n", f.MimeType)
		if f.ModifiedTime != "" {
			fmt.Fprintf(&sb, "  Modified: %s\n", f.ModifiedTime)
		}
		if f.Size > 0 {
			fmt.Fprintf(&sb, "  Size: %d bytes\n", f.Size)
		}
		sb.WriteString("\n")
	}

	if fileList.NextPageToken != "" {
		fmt.Fprintf(&sb, "Next page token: %s (more results available — use page_token to continue)\n", fileList.NextPageToken)
	}

	return sb.String()
}

// isTextMime returns true if the MIME type represents text content.
func isTextMime(mimeType string) bool {
	if strings.HasPrefix(mimeType, "text/") {
		return true
	}
	switch mimeType {
	case "application/json", "application/xml", "application/javascript",
		"application/yaml", "application/x-yaml",
		"application/toml",
		"application/x-sh", "application/x-shellscript",
		"application/sql",
		"application/typescript", "application/x-typescript":
		return true
	}
	if strings.HasSuffix(mimeType, "+json") || strings.HasSuffix(mimeType, "+xml") {
		return true
	}
	return false
}
