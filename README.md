# gdrive-readonly-mcp

A read-only Google Drive MCP server for Claude Desktop, written in Go. Provides secure, read-only access to Google Drive files, Google Docs, and Google Sheets through the Model Context Protocol.

## Features

- **List files** -- Browse files in Google Drive with optional name filtering and folder scoping
- **Search files** -- Full-text and name search across your entire Drive
- **File metadata** -- Retrieve detailed metadata for any file by ID
- **Read files** -- Extract text from Google Docs, read Google Sheets as TSV, or download text-based regular files (auto-detects type by MIME)
- **Read spreadsheets** -- Read specific cell ranges from Google Sheets using A1 notation (use this when you need a targeted range rather than the whole sheet)

## Security

This server is designed with security as a first principle:

- **Hardcoded read-only OAuth scopes** -- The server requests only `drive.readonly`, `documents.readonly`, and `spreadsheets.readonly` scopes. Write scopes are never used and cannot be configured.
- **Token file permissions** -- OAuth tokens are saved with `0600` permissions (owner read/write only).
- **Query escaping** -- All user input in Google Drive API queries is escaped to prevent injection.
- **File ID validation** -- File IDs are validated against a strict allowlist of characters before any API call.
- **Download size cap** -- Non-Google file downloads are capped at 1 MB to prevent memory exhaustion.
- **Binary file rejection** -- Only text-based MIME types are served; binary files return a clear error.
- **CSRF protection** -- The OAuth callback flow uses a cryptographically random state parameter.
- **No secrets in code** -- Credentials and tokens are loaded from external files, excluded from version control via `.gitignore`.

## Prerequisites

- Go 1.25.5 or later
- A Google Cloud project with the following APIs enabled:
  - Google Drive API
  - Google Docs API
  - Google Sheets API
- OAuth 2.0 credentials (Desktop application type)

## Setup

### 1. Create a Google Cloud project

Go to the [Google Cloud Console](https://console.cloud.google.com/) and create a new project (or select an existing one).

### 2. Enable APIs

In the Google Cloud Console, navigate to **APIs & Services > Library** and enable:

- Google Drive API
- Google Docs API
- Google Sheets API

### 3. Create OAuth 2.0 credentials

Navigate to **APIs & Services > Credentials** and create an OAuth 2.0 Client ID:

- Application type: **Desktop app**
- Name: any name you prefer (e.g., "gdrive-readonly-mcp")

### 4. Download credentials

Download the credentials JSON file and save it as `credentials.json` in the same directory as the built binary. Alternatively, set the `CREDENTIALS_FILE` environment variable to point to the file. For reliable path resolution (especially with symlinks), use the environment variables.

### 5. Build

```sh
go build -o gdrive-readonly-mcp .
```

Or use the Makefile:

```sh
make build
```

### 6. First run

```sh
./gdrive-readonly-mcp
```

On first run, the server will open your browser for Google OAuth consent. After you authorize, the token is saved to `token.json` for subsequent runs. The server communicates via stdin/stdout using the MCP JSON-RPC protocol, so it is intended to be launched by Claude Desktop (not run interactively).

### 7. Subsequent runs

On subsequent runs, the saved token is loaded automatically. If the token expires, it is refreshed and the updated token is persisted to disk.

## Claude Desktop Configuration

### macOS

Edit `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "gdrive-readonly": {
      "command": "/absolute/path/to/gdrive-readonly-mcp"
    }
  }
}
```

### Windows

Edit `%APPDATA%\Claude\claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "gdrive-readonly": {
      "command": "C:\\path\\to\\gdrive-readonly-mcp.exe",
      "env": {
        "CREDENTIALS_FILE": "C:\\path\\to\\credentials.json",
        "TOKEN_FILE": "C:\\path\\to\\token.json"
      }
    }
  }
}
```

If `credentials.json` and `token.json` are in the same directory as the binary, the `env` block can be omitted.

## Available Tools

| Tool | Description | Parameters |
|------|-------------|------------|
| `gdrive_list_files` | List files in Drive, optionally filtered by name or folder | `query` (opt), `folder_id` (opt), `page_size` (opt, default 20, max 100), `page_token` (opt) |
| `gdrive_search` | Search for files by name or content | `query` (required), `page_size` (opt, default 20, max 100), `page_token` (opt) |
| `gdrive_get_file_metadata` | Get detailed metadata for a file | `file_id` (required) |
| `gdrive_read_file` | Read a file's content -- auto-detects type: extracts text from Docs, reads all data from Sheets as TSV, or downloads text-based regular files | `file_id` (required) |
| `gdrive_read_sheet` | Read a specific range from a Google Sheet as TSV -- use this instead of `gdrive_read_file` when you need a targeted cell range | `file_id` (required), `range` (opt, A1 notation; reads entire first sheet if omitted) |

## Cross-Compilation

Build binaries for all supported platforms:

```sh
make all
```

This produces:

- `gdrive-readonly-mcp.exe` (Windows amd64)
- `gdrive-readonly-mcp-darwin-amd64` (macOS Intel)
- `gdrive-readonly-mcp-darwin-arm64` (macOS Apple Silicon)
- `gdrive-readonly-mcp-linux-amd64` (Linux amd64)

Individual platform targets are also available: `make windows`, `make darwin-amd64`, `make darwin-arm64`, `make linux`.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CREDENTIALS_FILE` | `credentials.json` in the binary's directory | Path to the Google OAuth credentials file |
| `TOKEN_FILE` | `token.json` in the binary's directory | Path to the saved OAuth token file |

## License

MIT
