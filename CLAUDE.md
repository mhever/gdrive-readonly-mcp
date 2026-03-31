# gdrive-readonly-mcp

Read-only Google Drive MCP server in Go.

## Build
go build -o gdrive-readonly-mcp .

## Test
go test ./...

## Cross-compile (Windows)
GOOS=windows GOARCH=amd64 go build -o gdrive-readonly-mcp.exe .

## Key conventions
- All OAuth scopes are hardcoded read-only — never add write scopes
- stdout is reserved for MCP JSON-RPC — all logging goes to stderr via log package
- User input in Drive queries must be escaped (single quotes, backslashes)
- File IDs must be validated before API calls
- Token files use 0600 permissions
- Use latest versions of all dependencies
