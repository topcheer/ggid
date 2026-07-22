package main

import (
	"fmt"
	"log"
	"os"

	"github.com/ggid/ggid/services/mcp/internal/client"
	"github.com/ggid/ggid/services/mcp/internal/server"
)

func main() {
	gatewayURL := os.Getenv("GGID_GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = "http://localhost:8080"
	}
	port := os.Getenv("GGID_MCP_PORT")
	if port == "" {
		port = "9060"
	}
	token := os.Getenv("GGID_ACCESS_TOKEN")
	if token == "" {
		log.Println("Warning: GGID_ACCESS_TOKEN not set — tools will fail until authenticated")
	}

	cli := client.New(gatewayURL, token, os.Getenv("GGID_TENANT_ID"))
	srv := server.New(cli)

	addr := ":" + port
	log.Printf("MCP Server starting on %s (gateway=%s)", addr, gatewayURL)
	if err := srv.ListenAndServe(addr); err != nil {
		fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
		os.Exit(1)
	}
}
