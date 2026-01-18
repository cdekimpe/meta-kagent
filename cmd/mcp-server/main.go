// Package main provides the entry point for the KMeta-Agent MCP server.
package main

import (
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/server"

	"github.com/kagent-dev/meta-kagent/internal/kubernetes"
	mcpserver "github.com/kagent-dev/meta-kagent/internal/server"
	"github.com/kagent-dev/meta-kagent/internal/tools"
)

func main() {
	// Get namespace from environment or default to "kagent"
	namespace := os.Getenv("KAGENT_NAMESPACE")
	if namespace == "" {
		namespace = "kagent"
	}

	// Initialize Kubernetes client
	k8sClient, err := kubernetes.NewClient(namespace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create Kubernetes client: %v\n", err)
		os.Exit(1)
	}

	// Create MCP server
	s := mcpserver.New(k8sClient)

	// Register all tools
	tools.RegisterAll(s)

	// Start server with stdio transport
	if err := server.ServeStdio(s.MCPServer()); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
