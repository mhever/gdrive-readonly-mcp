package main

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const serverVersion = "0.1.0"

func main() {
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "gdrive-readonly",
			Version: serverVersion,
		},
		nil,
	)

	registerTools(server)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
