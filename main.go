package main

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const serverVersion = "0.1.0"

func main() {
	credPath, err := resolveFilePath("CREDENTIALS_FILE", "credentials.json")
	if err != nil {
		log.Fatalf("Failed to resolve credentials path: %v", err)
	}

	tokenPath, err := resolveFilePath("TOKEN_FILE", "token.json")
	if err != nil {
		log.Fatalf("Failed to resolve token path: %v", err)
	}

	client, err := getOAuthClient(credPath, tokenPath)
	if err != nil {
		log.Fatalf("Failed to get OAuth client: %v", err)
	}

	ctx := context.Background()
	if err := initServices(ctx, client); err != nil {
		log.Fatalf("Failed to initialize Google API services: %v", err)
	}

	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "gdrive-readonly",
			Version: serverVersion,
		},
		nil,
	)

	registerTools(server)

	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Fatalf("MCP server error: %v", err)
	}
}
