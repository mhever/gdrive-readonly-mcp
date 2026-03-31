package main

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestHandleStatusContainsVersion(t *testing.T) {
	_, output, err := handleStatus(context.Background(), &mcp.CallToolRequest{}, statusInput{})
	if err != nil {
		t.Fatalf("handleStatus returned error: %v", err)
	}
	if !strings.Contains(output.Status, serverVersion) {
		t.Errorf("status output %q does not contain version %q", output.Status, serverVersion)
	}
}

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
