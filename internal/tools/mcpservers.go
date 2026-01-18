package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"sigs.k8s.io/yaml"

	"github.com/kagent-dev/meta-kagent/pkg/types"
)

// registerListMCPServers registers the list_mcp_servers tool.
func (ts *ToolServer) registerListMCPServers() {
	tool := mcp.NewTool("list_mcp_servers",
		mcp.WithDescription("List all MCPServer and RemoteMCPServer resources in the namespace."),
		mcp.WithBoolean("include_remote",
			mcp.Description("Include RemoteMCPServer resources (default: true)"),
		),
	)

	ts.server.AddTool(tool, ts.handleListMCPServers)
}

func (ts *ToolServer) handleListMCPServers(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	includeRemote := true
	if v, ok := req.Params.Arguments["include_remote"].(bool); ok {
		includeRemote = v
	}

	var result []map[string]interface{}

	// List MCPServers
	mcpServers, err := ts.k8sClient.ListMCPServers(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list MCP servers: %v", err)), nil
	}

	for _, server := range mcpServers {
		item := map[string]interface{}{
			"name":          server.Name,
			"namespace":     server.Namespace,
			"kind":          "MCPServer",
			"transportType": server.Spec.TransportType,
			"description":   server.Spec.Description,
		}
		if server.Spec.Deployment != nil {
			item["image"] = server.Spec.Deployment.Image
		}
		result = append(result, item)
	}

	// List RemoteMCPServers
	if includeRemote {
		remoteServers, err := ts.k8sClient.ListRemoteMCPServers(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list remote MCP servers: %v", err)), nil
		}

		for _, server := range remoteServers {
			item := map[string]interface{}{
				"name":        server.Name,
				"namespace":   server.Namespace,
				"kind":        "RemoteMCPServer",
				"url":         server.Spec.URL,
				"protocol":    server.Spec.Protocol,
				"description": server.Spec.Description,
			}
			result = append(result, item)
		}
	}

	if len(result) == 0 {
		return mcp.NewToolResultText("No MCP servers found in the namespace. Use create_mcp_server_manifest to create one."), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

// registerCreateMCPServerManifest registers the create_mcp_server_manifest tool.
func (ts *ToolServer) registerCreateMCPServerManifest() {
	tool := mcp.NewTool("create_mcp_server_manifest",
		mcp.WithDescription("Generate a new MCPServer or RemoteMCPServer manifest. MCPServer runs as a container with stdio transport; RemoteMCPServer connects to an external HTTP endpoint."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name for the MCP server resource"),
		),
		mcp.WithString("server_type",
			mcp.Required(),
			mcp.Description("Type: 'MCPServer' (local container with stdio) or 'RemoteMCPServer' (external HTTP endpoint)"),
		),
		mcp.WithString("description",
			mcp.Description("Human-readable description of the server's purpose"),
		),
		// MCPServer specific
		mcp.WithString("image",
			mcp.Description("Container image for MCPServer (required for MCPServer type)"),
		),
		mcp.WithString("command",
			mcp.Description("Command to run in the container"),
		),
		mcp.WithString("args_json",
			mcp.Description("JSON array of command arguments"),
		),
		mcp.WithNumber("port",
			mcp.Description("Container port (default: 3000)"),
		),
		// RemoteMCPServer specific
		mcp.WithString("url",
			mcp.Description("URL for RemoteMCPServer (required for RemoteMCPServer type)"),
		),
		mcp.WithString("protocol",
			mcp.Description("Protocol for RemoteMCPServer: 'STREAMABLE_HTTP' (default) or 'SSE'"),
		),
		mcp.WithString("timeout",
			mcp.Description("Request timeout (e.g., '30s', '5m')"),
		),
	)

	ts.server.AddTool(tool, ts.handleCreateMCPServerManifest)
}

func (ts *ToolServer) handleCreateMCPServerManifest(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, _ := req.Params.Arguments["name"].(string)
	serverType, _ := req.Params.Arguments["server_type"].(string)
	description, _ := req.Params.Arguments["description"].(string)

	if name == "" || serverType == "" {
		return mcp.NewToolResultError("name and server_type are required"), nil
	}

	if serverType == "MCPServer" {
		return ts.createMCPServerManifest(req, name, description)
	} else if serverType == "RemoteMCPServer" {
		return ts.createRemoteMCPServerManifest(req, name, description)
	}

	return mcp.NewToolResultError("server_type must be 'MCPServer' or 'RemoteMCPServer'"), nil
}

func (ts *ToolServer) createMCPServerManifest(req mcp.CallToolRequest, name, description string) (*mcp.CallToolResult, error) {
	image, _ := req.Params.Arguments["image"].(string)
	command, _ := req.Params.Arguments["command"].(string)
	argsJSON, _ := req.Params.Arguments["args_json"].(string)
	portFloat, _ := req.Params.Arguments["port"].(float64)

	if image == "" {
		return mcp.NewToolResultError("image is required for MCPServer type"), nil
	}

	port := int32(3000)
	if portFloat > 0 {
		port = int32(portFloat)
	}

	var args []string
	if argsJSON != "" {
		_ = json.Unmarshal([]byte(argsJSON), &args)
	}

	server := types.MCPServer{
		Spec: types.MCPServerSpec{
			Description: description,
			Deployment: &types.DeploymentSpec{
				Image: image,
				Cmd:   command,
				Args:  args,
				Port:  port,
			},
			TransportType:  "stdio",
			StdioTransport: map[string]interface{}{},
		},
	}
	server.APIVersion = "kagent.dev/v1alpha1"
	server.Kind = "MCPServer"
	server.Name = name
	server.Namespace = "kagent"

	output, _ := yaml.Marshal(server)

	result := fmt.Sprintf(`# Generated MCPServer Manifest
# This creates a local MCP server running as a container with stdio transport.
# Use validate_manifest to check, then apply_manifest to deploy.

%s`, string(output))

	return mcp.NewToolResultText(result), nil
}

func (ts *ToolServer) createRemoteMCPServerManifest(req mcp.CallToolRequest, name, description string) (*mcp.CallToolResult, error) {
	url, _ := req.Params.Arguments["url"].(string)
	protocol, _ := req.Params.Arguments["protocol"].(string)
	timeout, _ := req.Params.Arguments["timeout"].(string)

	if url == "" {
		return mcp.NewToolResultError("url is required for RemoteMCPServer type"), nil
	}

	if protocol == "" {
		protocol = "STREAMABLE_HTTP"
	}
	if timeout == "" {
		timeout = "30s"
	}

	server := types.RemoteMCPServer{
		Spec: types.RemoteMCPServerSpec{
			Description:      description,
			URL:              url,
			Protocol:         protocol,
			Timeout:          timeout,
			SSEReadTimeout:   "5m0s",
			TerminateOnClose: true,
		},
	}
	server.APIVersion = "kagent.dev/v1alpha2"
	server.Kind = "RemoteMCPServer"
	server.Name = name
	server.Namespace = "kagent"

	output, _ := yaml.Marshal(server)

	result := fmt.Sprintf(`# Generated RemoteMCPServer Manifest
# This connects to an external MCP server at %s using %s protocol.
# Use validate_manifest to check, then apply_manifest to deploy.

%s`, url, protocol, string(output))

	return mcp.NewToolResultText(result), nil
}
