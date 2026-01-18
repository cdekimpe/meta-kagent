// Package tools provides MCP tool implementations for managing kagent resources.
package tools

import (
	"github.com/kagent-dev/meta-kagent/internal/kubernetes"
	mcpserver "github.com/kagent-dev/meta-kagent/internal/server"
)

// ToolServer holds the dependencies for tool handlers.
type ToolServer struct {
	server    *mcpserver.Server
	k8sClient *kubernetes.Client
}

// RegisterAll registers all tools with the MCP server.
func RegisterAll(s *mcpserver.Server) {
	ts := &ToolServer{
		server:    s,
		k8sClient: s.K8sClient(),
	}

	// Discovery tools
	ts.registerListAgents()
	ts.registerGetAgent()
	ts.registerListModelConfigs()
	ts.registerListMCPServers()

	// Generation tools
	ts.registerCreateAgentManifest()
	ts.registerUpdateAgentManifest()
	ts.registerCreateModelConfigManifest()
	ts.registerCreateMCPServerManifest()
	ts.registerGenerateRBACManifest()

	// Validation and mutation tools
	ts.registerValidateManifest()
	ts.registerDiffManifest()
	ts.registerApplyManifest()
	ts.registerDeleteAgent()
}
