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

// registerListAgentSkills registers the list_agent_skills tool.
func (ts *ToolServer) registerListAgentSkills() {
	tool := mcp.NewTool("list_agent_skills",
		mcp.WithDescription("List A2A skills exposed by agents in the cluster. Shows which agents can be called by other agents via A2A protocol."),
		mcp.WithString("agent_name",
			mcp.Description("Filter to skills from a specific agent"),
		),
		mcp.WithString("tag",
			mcp.Description("Filter skills by tag (e.g., 'monitoring', 'kubernetes')"),
		),
	)

	ts.server.AddTool(tool, ts.handleListAgentSkills)
}

func (ts *ToolServer) handleListAgentSkills(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	agentName, _ := req.Params.Arguments["agent_name"].(string)
	tag, _ := req.Params.Arguments["tag"].(string)

	agents, err := ts.k8sClient.ListAgents(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list agents: %v", err)), nil
	}

	type skillInfo struct {
		AgentName   string   `json:"agentName"`
		SkillID     string   `json:"skillId"`
		SkillName   string   `json:"skillName"`
		Description string   `json:"description"`
		InputModes  []string `json:"inputModes,omitempty"`
		OutputModes []string `json:"outputModes,omitempty"`
		Tags        []string `json:"tags,omitempty"`
	}

	var results []skillInfo

	for _, agent := range agents {
		// Filter by agent name if specified
		if agentName != "" && agent.Name != agentName {
			continue
		}

		// Skip agents without A2A config
		a2aConfig := getA2AConfig(&agent)
		if a2aConfig == nil || len(a2aConfig.Skills) == 0 {
			continue
		}

		for _, skill := range a2aConfig.Skills {
			// Filter by tag if specified
			if tag != "" {
				hasTag := false
				for _, t := range skill.Tags {
					if strings.EqualFold(t, tag) {
						hasTag = true
						break
					}
				}
				if !hasTag {
					continue
				}
			}

			results = append(results, skillInfo{
				AgentName:   agent.Name,
				SkillID:     skill.ID,
				SkillName:   skill.Name,
				Description: skill.Description,
				InputModes:  skill.InputModes,
				OutputModes: skill.OutputModes,
				Tags:        skill.Tags,
			})
		}
	}

	if len(results) == 0 {
		if agentName != "" {
			return mcp.NewToolResultText(fmt.Sprintf("No A2A skills found for agent '%s'.", agentName)), nil
		}
		return mcp.NewToolResultText("No A2A skills found in any agents."), nil
	}

	output, _ := json.MarshalIndent(results, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

// registerDiscoverA2AAgents registers the discover_a2a_agents tool.
func (ts *ToolServer) registerDiscoverA2AAgents() {
	tool := mcp.NewTool("discover_a2a_agents",
		mcp.WithDescription("Discover agents in the cluster that expose A2A skills. Useful for finding agents that can be called by other agents."),
		mcp.WithString("skill_tag",
			mcp.Description("Filter to agents that have skills with this tag"),
		),
	)

	ts.server.AddTool(tool, ts.handleDiscoverA2AAgents)
}

func (ts *ToolServer) handleDiscoverA2AAgents(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	skillTag, _ := req.Params.Arguments["skill_tag"].(string)

	agents, err := ts.k8sClient.ListAgents(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list agents: %v", err)), nil
	}

	type agentInfo struct {
		Name        string   `json:"name"`
		Namespace   string   `json:"namespace"`
		Description string   `json:"description"`
		SkillCount  int      `json:"skillCount"`
		SkillIDs    []string `json:"skillIds"`
		Tags        []string `json:"allTags"`
	}

	var results []agentInfo

	for _, agent := range agents {
		// Skip agents without A2A config
		a2aConfig := getA2AConfig(&agent)
		if a2aConfig == nil || len(a2aConfig.Skills) == 0 {
			continue
		}

		// Collect all skill IDs and tags
		var skillIDs []string
		tagSet := make(map[string]bool)

		matchesTag := skillTag == ""

		for _, skill := range a2aConfig.Skills {
			skillIDs = append(skillIDs, skill.ID)
			for _, t := range skill.Tags {
				tagSet[t] = true
				if strings.EqualFold(t, skillTag) {
					matchesTag = true
				}
			}
		}

		if !matchesTag {
			continue
		}

		var allTags []string
		for t := range tagSet {
			allTags = append(allTags, t)
		}

		results = append(results, agentInfo{
			Name:        agent.Name,
			Namespace:   agent.Namespace,
			Description: agent.Spec.Description,
			SkillCount:  len(a2aConfig.Skills),
			SkillIDs:    skillIDs,
			Tags:        allTags,
		})
	}

	if len(results) == 0 {
		if skillTag != "" {
			return mcp.NewToolResultText(fmt.Sprintf("No A2A-enabled agents found with skill tag '%s'.", skillTag)), nil
		}
		return mcp.NewToolResultText("No A2A-enabled agents found in the cluster."), nil
	}

	output, _ := json.MarshalIndent(results, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

// registerGetAgentCard registers the get_agent_card tool.
func (ts *ToolServer) registerGetAgentCard() {
	tool := mcp.NewTool("get_agent_card",
		mcp.WithDescription("Generate an A2A Agent Card for an agent. The Agent Card is a JSON document that describes the agent's capabilities and skills for A2A communication."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the agent to generate the Agent Card for"),
		),
		mcp.WithString("endpoint_url",
			mcp.Description("Custom endpoint URL for the agent (defaults to Kubernetes service URL: http://<name>.<namespace>.svc.cluster.local)"),
		),
		mcp.WithString("output_format",
			mcp.Description("Output format: 'json' (default) or 'yaml'"),
		),
	)

	ts.server.AddTool(tool, ts.handleGetAgentCard)
}

func (ts *ToolServer) handleGetAgentCard(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, ok := req.Params.Arguments["name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}

	endpointURL, _ := req.Params.Arguments["endpoint_url"].(string)
	format := "json"
	if v, ok := req.Params.Arguments["output_format"].(string); ok && v != "" {
		format = v
	}

	agent, err := ts.k8sClient.GetAgent(ctx, name)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get agent: %v", err)), nil
	}

	// Generate default endpoint URL from Kubernetes service naming
	if endpointURL == "" {
		namespace := agent.Namespace
		if namespace == "" {
			namespace = "kagent"
		}
		endpointURL = fmt.Sprintf("http://%s.%s.svc.cluster.local", name, namespace)
	}

	// Build Agent Card
	card := types.AgentCard{
		AgentID:          name,
		Name:             name,
		Description:      agent.Spec.Description,
		URL:              endpointURL,
		ProtocolVersions: []string{"1.0"},
		Provider: &types.AgentProvider{
			Name: "kagent",
		},
		Capabilities: &types.AgentCapabilities{
			Streaming:         false,
			PushNotifications: false,
		},
		SecuritySchemes: map[string]types.SecurityScheme{
			"bearerAuth": {
				Type:        "http",
				Scheme:      "bearer",
				Description: "Bearer token authentication",
			},
		},
		Security: []string{"bearerAuth"},
	}

	// Add skills if present
	a2aConfig := getA2AConfig(agent)
	if a2aConfig != nil && len(a2aConfig.Skills) > 0 {
		card.Skills = a2aConfig.Skills
	}

	var output []byte
	if format == "yaml" {
		output, _ = yaml.Marshal(card)
	} else {
		output, _ = json.MarshalIndent(card, "", "  ")
	}

	result := fmt.Sprintf(`# A2A Agent Card for '%s'
# This Agent Card can be published for A2A discovery.
# URL: %s

%s`, name, endpointURL, string(output))

	return mcp.NewToolResultText(result), nil
}

// registerCreateSkillManifest registers the create_skill_manifest tool.
func (ts *ToolServer) registerCreateSkillManifest() {
	tool := mcp.NewTool("create_skill_manifest",
		mcp.WithDescription("Generate an A2A skill definition to add to an agent's a2aConfig. Skills define capabilities that other agents can invoke via A2A protocol."),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Unique skill identifier (use snake_case, e.g., 'analyze_logs')"),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Human-readable skill name (e.g., 'Log Analysis')"),
		),
		mcp.WithString("description",
			mcp.Required(),
			mcp.Description("Clear description of what this skill does and when to use it"),
		),
		mcp.WithString("input_modes",
			mcp.Description("Comma-separated input modes (e.g., 'text/plain,application/json'). Default: 'text/plain'"),
		),
		mcp.WithString("output_modes",
			mcp.Description("Comma-separated output modes (e.g., 'application/json,text/plain'). Default: 'text/plain'"),
		),
		mcp.WithString("tags",
			mcp.Description("Comma-separated tags for categorization (e.g., 'monitoring,logging')"),
		),
		mcp.WithString("examples",
			mcp.Description("Comma-separated usage examples (e.g., 'Analyze error logs,Find authentication issues')"),
		),
	)

	ts.server.AddTool(tool, ts.handleCreateSkillManifest)
}

func (ts *ToolServer) handleCreateSkillManifest(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, _ := req.Params.Arguments["id"].(string)
	name, _ := req.Params.Arguments["name"].(string)
	description, _ := req.Params.Arguments["description"].(string)
	inputModes, _ := req.Params.Arguments["input_modes"].(string)
	outputModes, _ := req.Params.Arguments["output_modes"].(string)
	tags, _ := req.Params.Arguments["tags"].(string)
	examples, _ := req.Params.Arguments["examples"].(string)

	if id == "" || name == "" || description == "" {
		return mcp.NewToolResultError("id, name, and description are required"), nil
	}

	skill := types.Skill{
		ID:          id,
		Name:        name,
		Description: description,
	}

	// Parse input modes
	if inputModes != "" {
		skill.InputModes = splitAndTrim(inputModes)
	} else {
		skill.InputModes = []string{"text/plain"}
	}

	// Parse output modes
	if outputModes != "" {
		skill.OutputModes = splitAndTrim(outputModes)
	} else {
		skill.OutputModes = []string{"text/plain"}
	}

	// Parse tags
	if tags != "" {
		skill.Tags = splitAndTrim(tags)
	}

	// Parse examples
	if examples != "" {
		skill.Examples = splitAndTrim(examples)
	}

	output, _ := yaml.Marshal(skill)

	result := fmt.Sprintf(`# A2A Skill Definition
# Add this skill to an agent's spec.a2aConfig.skills array.
# Use 'add_skill_to_agent' to add this skill to an existing agent.

%s

# JSON format for add_skill_to_agent:
# %s`, string(output), mustJSON(skill))

	return mcp.NewToolResultText(result), nil
}

// registerValidateSkill registers the validate_skill tool.
func (ts *ToolServer) registerValidateSkill() {
	tool := mcp.NewTool("validate_skill",
		mcp.WithDescription("Validate an A2A skill definition against the protocol specification. Checks required fields and best practices."),
		mcp.WithString("skill_json",
			mcp.Required(),
			mcp.Description("JSON representation of the skill to validate"),
		),
		mcp.WithBoolean("strict",
			mcp.Description("Enable strict validation including best practice checks (default: true)"),
		),
	)

	ts.server.AddTool(tool, ts.handleValidateSkill)
}

func (ts *ToolServer) handleValidateSkill(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	skillJSON, _ := req.Params.Arguments["skill_json"].(string)
	if skillJSON == "" {
		return mcp.NewToolResultError("skill_json is required"), nil
	}

	strict := true
	if v, ok := req.Params.Arguments["strict"].(bool); ok {
		strict = v
	}

	var skill types.Skill
	if err := json.Unmarshal([]byte(skillJSON), &skill); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid JSON: %v", err)), nil
	}

	type issue struct {
		Severity string `json:"severity"` // "error" or "warning"
		Field    string `json:"field"`
		Message  string `json:"message"`
	}

	var issues []issue

	// Required field validation
	if skill.ID == "" {
		issues = append(issues, issue{
			Severity: "error",
			Field:    "id",
			Message:  "skill id is required",
		})
	}
	if skill.Name == "" {
		issues = append(issues, issue{
			Severity: "error",
			Field:    "name",
			Message:  "skill name is required",
		})
	}
	if skill.Description == "" {
		issues = append(issues, issue{
			Severity: "error",
			Field:    "description",
			Message:  "skill description is required",
		})
	}

	// Strict validation (best practices)
	if strict {
		if len(skill.Description) < 20 {
			issues = append(issues, issue{
				Severity: "warning",
				Field:    "description",
				Message:  "description is short; consider providing more detail for A2A discovery",
			})
		}
		if len(skill.Examples) == 0 {
			issues = append(issues, issue{
				Severity: "warning",
				Field:    "examples",
				Message:  "consider adding examples to help other agents understand how to use this skill",
			})
		}
		if len(skill.Tags) == 0 {
			issues = append(issues, issue{
				Severity: "warning",
				Field:    "tags",
				Message:  "consider adding tags to improve skill discoverability",
			})
		}
		if len(skill.InputModes) == 0 {
			issues = append(issues, issue{
				Severity: "warning",
				Field:    "inputModes",
				Message:  "consider specifying input modes (e.g., 'text/plain', 'application/json')",
			})
		}
		if len(skill.OutputModes) == 0 {
			issues = append(issues, issue{
				Severity: "warning",
				Field:    "outputModes",
				Message:  "consider specifying output modes",
			})
		}
	}

	// Count errors
	errorCount := 0
	warningCount := 0
	for _, i := range issues {
		if i.Severity == "error" {
			errorCount++
		} else {
			warningCount++
		}
	}

	if len(issues) == 0 {
		return mcp.NewToolResultText("✓ Skill validation passed. No issues found."), nil
	}

	output, _ := json.MarshalIndent(issues, "", "  ")
	summary := fmt.Sprintf("# Skill Validation Results\n# Errors: %d, Warnings: %d\n\n%s", errorCount, warningCount, string(output))

	if errorCount > 0 {
		return mcp.NewToolResultText(summary + "\n\n⚠ Validation failed with errors. Fix the errors before using this skill."), nil
	}

	return mcp.NewToolResultText(summary + "\n\n✓ Validation passed with warnings. Consider addressing the warnings."), nil
}

// registerAddSkillToAgent registers the add_skill_to_agent tool.
func (ts *ToolServer) registerAddSkillToAgent() {
	tool := mcp.NewTool("add_skill_to_agent",
		mcp.WithDescription("Generate an updated agent manifest with a new A2A skill added. Returns manifest for review before applying."),
		mcp.WithString("agent_name",
			mcp.Required(),
			mcp.Description("Name of the agent to add the skill to"),
		),
		mcp.WithString("skill_json",
			mcp.Required(),
			mcp.Description("JSON representation of the skill to add"),
		),
	)

	ts.server.AddTool(tool, ts.handleAddSkillToAgent)
}

func (ts *ToolServer) handleAddSkillToAgent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	agentName, _ := req.Params.Arguments["agent_name"].(string)
	skillJSON, _ := req.Params.Arguments["skill_json"].(string)

	if agentName == "" || skillJSON == "" {
		return mcp.NewToolResultError("agent_name and skill_json are required"), nil
	}

	// Parse skill
	var skill types.Skill
	if err := json.Unmarshal([]byte(skillJSON), &skill); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid skill JSON: %v", err)), nil
	}

	// Validate skill has required fields
	if skill.ID == "" || skill.Name == "" || skill.Description == "" {
		return mcp.NewToolResultError("skill must have id, name, and description"), nil
	}

	// Get existing agent
	agent, err := ts.k8sClient.GetAgent(ctx, agentName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get agent: %v", err)), nil
	}

	// Initialize A2AConfig if not present
	a2aConfig := getA2AConfig(agent)
	if a2aConfig == nil {
		a2aConfig = &types.A2AConfig{}
		setA2AConfig(agent, a2aConfig)
	}

	// Check for duplicate skill ID
	for _, existing := range a2aConfig.Skills {
		if existing.ID == skill.ID {
			return mcp.NewToolResultError(fmt.Sprintf("Skill with ID '%s' already exists on agent '%s'", skill.ID, agentName)), nil
		}
	}

	// Add the skill
	a2aConfig.Skills = append(a2aConfig.Skills, skill)

	// Set proper TypeMeta
	agent.APIVersion = "kagent.dev/v1alpha2"
	agent.Kind = "Agent"

	output, _ := yaml.Marshal(agent)

	result := fmt.Sprintf(`# Updated Agent Manifest
# IMPORTANT: Review the changes before applying.
# The skill '%s' has been added to the agent's a2aConfig.
# Use diff_manifest to see changes, then apply_manifest to deploy.

%s`, skill.Name, string(output))

	return mcp.NewToolResultText(result), nil
}

// registerRemoveSkillFromAgent registers the remove_skill_from_agent tool.
func (ts *ToolServer) registerRemoveSkillFromAgent() {
	tool := mcp.NewTool("remove_skill_from_agent",
		mcp.WithDescription("Generate an updated agent manifest with an A2A skill removed. Returns manifest for review before applying."),
		mcp.WithString("agent_name",
			mcp.Required(),
			mcp.Description("Name of the agent to remove the skill from"),
		),
		mcp.WithString("skill_id",
			mcp.Required(),
			mcp.Description("ID of the skill to remove"),
		),
	)

	ts.server.AddTool(tool, ts.handleRemoveSkillFromAgent)
}

func (ts *ToolServer) handleRemoveSkillFromAgent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	agentName, _ := req.Params.Arguments["agent_name"].(string)
	skillID, _ := req.Params.Arguments["skill_id"].(string)

	if agentName == "" || skillID == "" {
		return mcp.NewToolResultError("agent_name and skill_id are required"), nil
	}

	// Get existing agent
	agent, err := ts.k8sClient.GetAgent(ctx, agentName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get agent: %v", err)), nil
	}

	// Check if agent has A2A config
	a2aConfig := getA2AConfig(agent)
	if a2aConfig == nil || len(a2aConfig.Skills) == 0 {
		return mcp.NewToolResultError(fmt.Sprintf("Agent '%s' has no A2A skills configured", agentName)), nil
	}

	// Find and remove the skill
	found := false
	var filteredSkills []types.Skill
	for _, skill := range a2aConfig.Skills {
		if skill.ID == skillID {
			found = true
		} else {
			filteredSkills = append(filteredSkills, skill)
		}
	}

	if !found {
		return mcp.NewToolResultError(fmt.Sprintf("Skill with ID '%s' not found on agent '%s'", skillID, agentName)), nil
	}

	a2aConfig.Skills = filteredSkills

	// Set proper TypeMeta
	agent.APIVersion = "kagent.dev/v1alpha2"
	agent.Kind = "Agent"

	output, _ := yaml.Marshal(agent)

	result := fmt.Sprintf(`# Updated Agent Manifest
# IMPORTANT: Review the changes before applying.
# The skill '%s' has been removed from the agent's a2aConfig.
# Use diff_manifest to see changes, then apply_manifest to deploy.

%s`, skillID, string(output))

	return mcp.NewToolResultText(result), nil
}

// Helper functions

// getA2AConfig returns the A2AConfig from an agent, checking both
// spec.declarative.a2aConfig (kagent format) and spec.a2aConfig (legacy).
func getA2AConfig(agent *types.Agent) *types.A2AConfig {
	// Check declarative config first (kagent's actual location)
	if agent.Spec.Declarative != nil && agent.Spec.Declarative.A2AConfig != nil {
		return agent.Spec.Declarative.A2AConfig
	}
	// Fallback to spec-level config
	return agent.Spec.A2AConfig
}

// setA2AConfig sets the A2AConfig on an agent in the declarative spec.
func setA2AConfig(agent *types.Agent, config *types.A2AConfig) {
	if agent.Spec.Declarative == nil {
		agent.Spec.Declarative = &types.DeclarativeSpec{}
	}
	agent.Spec.Declarative.A2AConfig = config
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func mustJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
