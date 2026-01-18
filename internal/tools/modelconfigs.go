package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"sigs.k8s.io/yaml"

	"github.com/kagent-dev/meta-kagent/pkg/types"
)

// registerListModelConfigs registers the list_model_configs tool.
func (ts *ToolServer) registerListModelConfigs() {
	tool := mcp.NewTool("list_model_configs",
		mcp.WithDescription("List all kagent ModelConfig resources in the namespace. Returns provider, model, and secret reference for each."),
	)

	ts.server.AddTool(tool, ts.handleListModelConfigs)
}

func (ts *ToolServer) handleListModelConfigs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	configs, err := ts.k8sClient.ListModelConfigs(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list model configs: %v", err)), nil
	}

	if len(configs) == 0 {
		return mcp.NewToolResultText("No ModelConfigs found in the namespace. Use create_model_config_manifest to create one."), nil
	}

	var result []map[string]interface{}
	for _, config := range configs {
		item := map[string]interface{}{
			"name":         config.Name,
			"namespace":    config.Namespace,
			"provider":     config.Spec.Provider,
			"model":        config.Spec.Model,
			"apiKeySecret": config.Spec.APIKeySecret,
		}
		result = append(result, item)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

// registerCreateModelConfigManifest registers the create_model_config_manifest tool.
func (ts *ToolServer) registerCreateModelConfigManifest() {
	tool := mcp.NewTool("create_model_config_manifest",
		mcp.WithDescription("Generate a new ModelConfig manifest for LLM provider configuration. Returns YAML for review before applying."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name for the ModelConfig resource"),
		),
		mcp.WithString("provider",
			mcp.Required(),
			mcp.Description("LLM provider: OpenAI, AzureOpenAI, Anthropic, Gemini, Ollama, or Custom"),
		),
		mcp.WithString("model",
			mcp.Required(),
			mcp.Description("Model identifier (e.g., gpt-4o, claude-sonnet-4-20250514, gemini-2.5-pro)"),
		),
		mcp.WithString("api_key_secret",
			mcp.Required(),
			mcp.Description("Name of Kubernetes Secret containing the API key"),
		),
		mcp.WithString("api_key_secret_key",
			mcp.Description("Key within the secret that holds the API key (default varies by provider)"),
		),
		mcp.WithString("base_url",
			mcp.Description("Custom base URL for the API (for Custom provider or proxies)"),
		),
	)

	ts.server.AddTool(tool, ts.handleCreateModelConfigManifest)
}

func (ts *ToolServer) handleCreateModelConfigManifest(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, _ := req.Params.Arguments["name"].(string)
	provider, _ := req.Params.Arguments["provider"].(string)
	model, _ := req.Params.Arguments["model"].(string)
	apiKeySecret, _ := req.Params.Arguments["api_key_secret"].(string)
	apiKeySecretKey, _ := req.Params.Arguments["api_key_secret_key"].(string)
	baseURL, _ := req.Params.Arguments["base_url"].(string)

	if name == "" || provider == "" || model == "" || apiKeySecret == "" {
		return mcp.NewToolResultError("name, provider, model, and api_key_secret are required"), nil
	}

	// Validate provider
	validProviders := map[string]bool{
		"OpenAI":      true,
		"AzureOpenAI": true,
		"Anthropic":   true,
		"Gemini":      true,
		"Ollama":      true,
		"Custom":      true,
	}
	if !validProviders[provider] {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid provider '%s'. Must be one of: OpenAI, AzureOpenAI, Anthropic, Gemini, Ollama, Custom", provider)), nil
	}

	// Set default secret key based on provider
	if apiKeySecretKey == "" {
		switch provider {
		case "OpenAI":
			apiKeySecretKey = "OPENAI_API_KEY"
		case "Anthropic":
			apiKeySecretKey = "ANTHROPIC_API_KEY"
		case "Gemini":
			apiKeySecretKey = "GOOGLE_API_KEY"
		case "AzureOpenAI":
			apiKeySecretKey = "AZURE_OPENAI_API_KEY"
		default:
			apiKeySecretKey = "API_KEY"
		}
	}

	config := types.ModelConfig{
		Spec: types.ModelConfigSpec{
			Provider:        provider,
			Model:           model,
			APIKeySecret:    apiKeySecret,
			APIKeySecretKey: apiKeySecretKey,
			BaseURL:         baseURL,
		},
	}
	config.APIVersion = "kagent.dev/v1alpha2"
	config.Kind = "ModelConfig"
	config.Name = name
	config.Namespace = "kagent"

	// Add provider-specific empty config
	switch provider {
	case "OpenAI":
		config.Spec.OpenAI = map[string]interface{}{}
	case "Anthropic":
		config.Spec.Anthropic = map[string]interface{}{}
	case "Gemini":
		config.Spec.Gemini = map[string]interface{}{}
	case "AzureOpenAI":
		config.Spec.Azure = map[string]interface{}{}
	case "Ollama":
		config.Spec.Ollama = map[string]interface{}{}
	}

	output, _ := yaml.Marshal(config)

	result := fmt.Sprintf(`# Generated ModelConfig Manifest
# IMPORTANT: Ensure the Kubernetes Secret '%s' exists with key '%s' containing the API key.
# Use validate_manifest to check, then apply_manifest to deploy.

%s`, apiKeySecret, apiKeySecretKey, string(output))

	return mcp.NewToolResultText(result), nil
}
