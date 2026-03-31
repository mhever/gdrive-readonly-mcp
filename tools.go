package main

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Placeholder tool to verify the MCP server works.
// Will be replaced with real Drive/Docs/Sheets tools.

type statusInput struct{}

type statusOutput struct {
	Status string `json:"status"`
}

func handleStatus(ctx context.Context, req *mcp.CallToolRequest, input statusInput) (*mcp.CallToolResult, statusOutput, error) {
	return nil, statusOutput{Status: "Google Drive MCP Server is running (v" + serverVersion + ")"}, nil
}

func registerTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "gdrive_status",
		Description: "Check if the Google Drive MCP server is running",
	}, handleStatus)
}
