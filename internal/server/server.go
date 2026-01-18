// Package server provides the MCP server implementation.
package server

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/kagent-dev/meta-kagent/internal/kubernetes"
)

// Server wraps the MCP server with kagent-specific functionality.
type Server struct {
	mcpServer *server.MCPServer
	k8sClient *kubernetes.Client
}

// New creates a new MCP server for the meta-kagent.
func New(k8sClient *kubernetes.Client) *Server {
	mcpServer := server.NewMCPServer(
		"kmeta-agent-tools",
		"1.0.0",
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
	)

	return &Server{
		mcpServer: mcpServer,
		k8sClient: k8sClient,
	}
}

// MCPServer returns the underlying MCP server.
func (s *Server) MCPServer() *server.MCPServer {
	return s.mcpServer
}

// K8sClient returns the Kubernetes client.
func (s *Server) K8sClient() *kubernetes.Client {
	return s.k8sClient
}

// AddTool is a convenience wrapper for adding tools.
func (s *Server) AddTool(tool mcp.Tool, handler server.ToolHandlerFunc) {
	s.mcpServer.AddTool(tool, handler)
}
