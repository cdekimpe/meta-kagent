package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"sigs.k8s.io/yaml"

	"github.com/kagent-dev/meta-kagent/pkg/types"
)

// registerListAgents registers the list_agents tool.
func (ts *ToolServer) registerListAgents() {
	tool := mcp.NewTool("list_agents",
		mcp.WithDescription("List all kagent Agents in the namespace. Returns name, description, type, and status for each agent."),
		mcp.WithBoolean("include_status",
			mcp.Description("Include status information (ready, accepted) in the output"),
		),
	)

	ts.server.AddTool(tool, ts.handleListAgents)
}

func (ts *ToolServer) handleListAgents(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	includeStatus := false
	if v, ok := req.Params.Arguments["include_status"].(bool); ok {
		includeStatus = v
	}

	agents, err := ts.k8sClient.ListAgents(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list agents: %v", err)), nil
	}

	if len(agents) == 0 {
		return mcp.NewToolResultText("No agents found in the namespace."), nil
	}

	var result []map[string]interface{}
	for _, agent := range agents {
		item := map[string]interface{}{
			"name":        agent.Name,
			"namespace":   agent.Namespace,
			"type":        agent.Spec.Type,
			"description": agent.Spec.Description,
		}
		if agent.Spec.Declarative != nil {
			item["modelConfig"] = agent.Spec.Declarative.ModelConfig
			item["toolCount"] = len(agent.Spec.Declarative.Tools)
		}
		if includeStatus {
			item["ready"] = agent.Status.Ready
			item["accepted"] = agent.Status.Accepted
		}
		result = append(result, item)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

// registerGetAgent registers the get_agent tool.
func (ts *ToolServer) registerGetAgent() {
	tool := mcp.NewTool("get_agent",
		mcp.WithDescription("Get detailed information about a specific kagent Agent including its full specification."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the agent to retrieve"),
		),
		mcp.WithString("output_format",
			mcp.Description("Output format: 'yaml' (default) or 'json'"),
		),
	)

	ts.server.AddTool(tool, ts.handleGetAgent)
}

func (ts *ToolServer) handleGetAgent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, ok := req.Params.Arguments["name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}

	format := "yaml"
	if v, ok := req.Params.Arguments["output_format"].(string); ok && v != "" {
		format = v
	}

	agent, err := ts.k8sClient.GetAgent(ctx, name)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get agent: %v", err)), nil
	}

	// Set proper TypeMeta for output
	agent.APIVersion = "kagent.dev/v1alpha2"
	agent.Kind = "Agent"

	var output []byte
	if format == "json" {
		output, _ = json.MarshalIndent(agent, "", "  ")
	} else {
		output, _ = yaml.Marshal(agent)
	}

	return mcp.NewToolResultText(string(output)), nil
}

// registerCreateAgentManifest registers the create_agent_manifest tool.
func (ts *ToolServer) registerCreateAgentManifest() {
	tool := mcp.NewTool("create_agent_manifest",
		mcp.WithDescription("Generate a new kagent Agent manifest. Returns YAML that should be reviewed and validated before applying."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name for the new agent"),
		),
		mcp.WithString("description",
			mcp.Required(),
			mcp.Description("Human-readable description of what the agent does"),
		),
		mcp.WithString("system_message",
			mcp.Required(),
			mcp.Description("The system prompt that defines the agent's behavior, capabilities, and constraints"),
		),
		mcp.WithString("model_config",
			mcp.Required(),
			mcp.Description("Name of the ModelConfig resource to use for LLM configuration"),
		),
		mcp.WithString("tools_json",
			mcp.Description("JSON array of tool configurations. Format: [{\"mcpServer\": \"server-name\", \"kind\": \"MCPServer\", \"tools\": [\"tool1\", \"tool2\"]}]"),
		),
		mcp.WithString("skills_json",
			mcp.Description("JSON array of A2A skill configurations. Format: [{\"id\": \"skill-id\", \"name\": \"Skill Name\", \"description\": \"...\"}]"),
		),
	)

	ts.server.AddTool(tool, ts.handleCreateAgentManifest)
}

func (ts *ToolServer) handleCreateAgentManifest(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, _ := req.Params.Arguments["name"].(string)
	description, _ := req.Params.Arguments["description"].(string)
	systemMessage, _ := req.Params.Arguments["system_message"].(string)
	modelConfig, _ := req.Params.Arguments["model_config"].(string)
	toolsJSON, _ := req.Params.Arguments["tools_json"].(string)
	skillsJSON, _ := req.Params.Arguments["skills_json"].(string)

	if name == "" || systemMessage == "" || modelConfig == "" {
		return mcp.NewToolResultError("name, system_message, and model_config are required"), nil
	}

	// Build agent manifest
	agent := types.Agent{
		Spec: types.AgentSpec{
			Type:        "Declarative",
			Description: description,
			Declarative: &types.DeclarativeSpec{
				ModelConfig:   modelConfig,
				SystemMessage: systemMessage,
			},
		},
	}
	agent.APIVersion = "kagent.dev/v1alpha2"
	agent.Kind = "Agent"
	agent.Name = name
	agent.Namespace = "kagent"

	// Parse tools if provided
	if toolsJSON != "" {
		var toolConfigs []struct {
			MCPServer string   `json:"mcpServer"`
			Kind      string   `json:"kind"`
			Tools     []string `json:"tools"`
		}
		if err := json.Unmarshal([]byte(toolsJSON), &toolConfigs); err == nil {
			for _, tc := range toolConfigs {
				kind := tc.Kind
				if kind == "" {
					kind = "MCPServer"
				}
				agent.Spec.Declarative.Tools = append(agent.Spec.Declarative.Tools, types.ToolSpec{
					Type: "McpServer",
					McpServer: &types.McpServerRef{
						Name:      tc.MCPServer,
						Kind:      kind,
						ToolNames: tc.Tools,
					},
				})
			}
		}
	}

	// Parse skills if provided
	if skillsJSON != "" {
		var skills []types.Skill
		if err := json.Unmarshal([]byte(skillsJSON), &skills); err == nil {
			agent.Spec.A2AConfig = &types.A2AConfig{
				Skills: skills,
			}
		}
	}

	output, _ := yaml.Marshal(agent)

	result := fmt.Sprintf(`# Generated Agent Manifest
# IMPORTANT: Review this manifest carefully before applying.
# Use validate_manifest to check for issues, then apply_manifest to deploy.

%s`, string(output))

	return mcp.NewToolResultText(result), nil
}

// registerUpdateAgentManifest registers the update_agent_manifest tool.
func (ts *ToolServer) registerUpdateAgentManifest() {
	tool := mcp.NewTool("update_agent_manifest",
		mcp.WithDescription("Generate an updated manifest for an existing Agent. Fetches current state and applies the specified modifications."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the agent to update"),
		),
		mcp.WithString("system_message",
			mcp.Description("New system prompt (optional, keeps current if not provided)"),
		),
		mcp.WithString("description",
			mcp.Description("New description (optional)"),
		),
		mcp.WithString("model_config",
			mcp.Description("New ModelConfig reference (optional)"),
		),
		mcp.WithString("add_tools_json",
			mcp.Description("JSON array of tools to add. Format: [{\"mcpServer\": \"name\", \"kind\": \"MCPServer\", \"tools\": [\"tool1\"]}]"),
		),
		mcp.WithString("remove_tool_servers",
			mcp.Description("Comma-separated list of MCP server names to remove from the agent"),
		),
	)

	ts.server.AddTool(tool, ts.handleUpdateAgentManifest)
}

func (ts *ToolServer) handleUpdateAgentManifest(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, _ := req.Params.Arguments["name"].(string)
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}

	// Get current agent
	agent, err := ts.k8sClient.GetAgent(ctx, name)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get agent: %v", err)), nil
	}

	// Apply updates
	if v, ok := req.Params.Arguments["description"].(string); ok && v != "" {
		agent.Spec.Description = v
	}

	if agent.Spec.Declarative != nil {
		if v, ok := req.Params.Arguments["system_message"].(string); ok && v != "" {
			agent.Spec.Declarative.SystemMessage = v
		}
		if v, ok := req.Params.Arguments["model_config"].(string); ok && v != "" {
			agent.Spec.Declarative.ModelConfig = v
		}
	}

	// Remove tools
	if removeServers, ok := req.Params.Arguments["remove_tool_servers"].(string); ok && removeServers != "" {
		serverNames := strings.Split(removeServers, ",")
		removeMap := make(map[string]bool)
		for _, s := range serverNames {
			removeMap[strings.TrimSpace(s)] = true
		}

		if agent.Spec.Declarative != nil {
			var filteredTools []types.ToolSpec
			for _, tool := range agent.Spec.Declarative.Tools {
				if tool.McpServer == nil || !removeMap[tool.McpServer.Name] {
					filteredTools = append(filteredTools, tool)
				}
			}
			agent.Spec.Declarative.Tools = filteredTools
		}
	}

	// Add tools
	if addToolsJSON, ok := req.Params.Arguments["add_tools_json"].(string); ok && addToolsJSON != "" {
		var toolConfigs []struct {
			MCPServer string   `json:"mcpServer"`
			Kind      string   `json:"kind"`
			Tools     []string `json:"tools"`
		}
		if err := json.Unmarshal([]byte(addToolsJSON), &toolConfigs); err == nil && agent.Spec.Declarative != nil {
			for _, tc := range toolConfigs {
				kind := tc.Kind
				if kind == "" {
					kind = "MCPServer"
				}
				agent.Spec.Declarative.Tools = append(agent.Spec.Declarative.Tools, types.ToolSpec{
					Type: "McpServer",
					McpServer: &types.McpServerRef{
						Name:      tc.MCPServer,
						Kind:      kind,
						ToolNames: tc.Tools,
					},
				})
			}
		}
	}

	// Set proper TypeMeta
	agent.APIVersion = "kagent.dev/v1alpha2"
	agent.Kind = "Agent"

	output, _ := yaml.Marshal(agent)

	result := fmt.Sprintf(`# Updated Agent Manifest
# IMPORTANT: Review the changes before applying.
# Use diff_manifest to see changes, then apply_manifest to deploy.

%s`, string(output))

	return mcp.NewToolResultText(result), nil
}

// registerDeleteAgent registers the delete_agent tool.
func (ts *ToolServer) registerDeleteAgent() {
	tool := mcp.NewTool("delete_agent",
		mcp.WithDescription("Delete a kagent Agent from the cluster. IMPORTANT: This action is destructive. Use dry_run=true to preview without deleting."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the agent to delete"),
		),
		mcp.WithBoolean("dry_run",
			mcp.Description("If true, only simulate the deletion without actually removing the agent"),
		),
	)

	ts.server.AddTool(tool, ts.handleDeleteAgent)
}

func (ts *ToolServer) handleDeleteAgent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, _ := req.Params.Arguments["name"].(string)
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}

	dryRun := false
	if v, ok := req.Params.Arguments["dry_run"].(bool); ok {
		dryRun = v
	}

	// Verify agent exists first
	agent, err := ts.k8sClient.GetAgent(ctx, name)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Agent not found: %v", err)), nil
	}

	if dryRun {
		return mcp.NewToolResultText(fmt.Sprintf(`# Dry Run: Delete Agent

The following agent would be deleted:
- Name: %s
- Namespace: %s
- Description: %s

To actually delete, call delete_agent with dry_run=false.`,
			agent.Name, agent.Namespace, agent.Spec.Description)), nil
	}

	err = ts.k8sClient.Delete(ctx, "Agent", name, false)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete agent: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully deleted agent '%s'.", name)), nil
}
