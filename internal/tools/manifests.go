package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/mark3labs/mcp-go/mcp"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

// registerValidateManifest registers the validate_manifest tool.
func (ts *ToolServer) registerValidateManifest() {
	tool := mcp.NewTool("validate_manifest",
		mcp.WithDescription("Validate a kagent manifest for correctness and completeness. Checks required fields, references, and best practices."),
		mcp.WithString("manifest",
			mcp.Required(),
			mcp.Description("YAML manifest to validate"),
		),
		mcp.WithBoolean("strict",
			mcp.Description("Enable strict validation including best practice checks (default: true)"),
		),
	)

	ts.server.AddTool(tool, ts.handleValidateManifest)
}

func (ts *ToolServer) handleValidateManifest(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	manifest, _ := req.Params.Arguments["manifest"].(string)
	if manifest == "" {
		return mcp.NewToolResultError("manifest is required"), nil
	}

	strict := true
	if v, ok := req.Params.Arguments["strict"].(bool); ok {
		strict = v
	}

	// Parse manifest
	var obj unstructured.Unstructured
	if err := yaml.Unmarshal([]byte(manifest), &obj.Object); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse manifest: %v", err)), nil
	}

	var issues []ValidationIssue

	// Basic validation
	if obj.GetAPIVersion() == "" {
		issues = append(issues, ValidationIssue{
			Severity: "error",
			Field:    "apiVersion",
			Message:  "apiVersion is required",
		})
	}

	if obj.GetKind() == "" {
		issues = append(issues, ValidationIssue{
			Severity: "error",
			Field:    "kind",
			Message:  "kind is required",
		})
	}

	if obj.GetName() == "" {
		issues = append(issues, ValidationIssue{
			Severity: "error",
			Field:    "metadata.name",
			Message:  "metadata.name is required",
		})
	}

	// Kind-specific validation
	switch obj.GetKind() {
	case "Agent":
		issues = append(issues, ts.validateAgent(ctx, &obj, strict)...)
	case "ModelConfig":
		issues = append(issues, ts.validateModelConfig(ctx, &obj, strict)...)
	case "MCPServer":
		issues = append(issues, ts.validateMCPServer(ctx, &obj, strict)...)
	case "RemoteMCPServer":
		issues = append(issues, ts.validateRemoteMCPServer(ctx, &obj, strict)...)
	default:
		issues = append(issues, ValidationIssue{
			Severity: "warning",
			Field:    "kind",
			Message:  fmt.Sprintf("Unknown kind '%s'. Expected: Agent, ModelConfig, MCPServer, or RemoteMCPServer", obj.GetKind()),
		})
	}

	// Format result
	if len(issues) == 0 {
		return mcp.NewToolResultText("✓ Validation passed. Manifest is valid and ready to apply."), nil
	}

	var result strings.Builder
	result.WriteString("Validation Results:\n\n")

	hasErrors := false
	for _, issue := range issues {
		prefix := "⚠️  WARNING"
		if issue.Severity == "error" {
			prefix = "❌ ERROR"
			hasErrors = true
		}
		result.WriteString(fmt.Sprintf("%s [%s]: %s\n", prefix, issue.Field, issue.Message))
	}

	result.WriteString("\n")
	if hasErrors {
		result.WriteString("❌ Manifest has errors and should not be applied until they are resolved.")
	} else {
		result.WriteString("⚠️  Manifest has warnings but can be applied. Consider addressing warnings for best practices.")
	}

	return mcp.NewToolResultText(result.String()), nil
}

// ValidationIssue represents a validation error or warning.
type ValidationIssue struct {
	Severity string `json:"severity"` // "error" or "warning"
	Field    string `json:"field"`
	Message  string `json:"message"`
}

func (ts *ToolServer) validateAgent(ctx context.Context, obj *unstructured.Unstructured, strict bool) []ValidationIssue {
	var issues []ValidationIssue

	// Check spec.type
	specType, found, _ := unstructured.NestedString(obj.Object, "spec", "type")
	if !found || specType == "" {
		issues = append(issues, ValidationIssue{
			Severity: "error",
			Field:    "spec.type",
			Message:  "spec.type is required (should be 'Declarative' or 'BYO')",
		})
	}

	if specType == "Declarative" {
		// Check modelConfig reference
		modelConfig, found, _ := unstructured.NestedString(obj.Object, "spec", "declarative", "modelConfig")
		if !found || modelConfig == "" {
			issues = append(issues, ValidationIssue{
				Severity: "error",
				Field:    "spec.declarative.modelConfig",
				Message:  "spec.declarative.modelConfig is required for Declarative agents",
			})
		} else {
			// Verify ModelConfig exists
			_, err := ts.k8sClient.GetModelConfig(ctx, modelConfig)
			if err != nil {
				issues = append(issues, ValidationIssue{
					Severity: "warning",
					Field:    "spec.declarative.modelConfig",
					Message:  fmt.Sprintf("ModelConfig '%s' not found in namespace. Ensure it exists before applying.", modelConfig),
				})
			}
		}

		// Check systemMessage
		systemMessage, found, _ := unstructured.NestedString(obj.Object, "spec", "declarative", "systemMessage")
		if !found || systemMessage == "" {
			issues = append(issues, ValidationIssue{
				Severity: "error",
				Field:    "spec.declarative.systemMessage",
				Message:  "spec.declarative.systemMessage is required for Declarative agents",
			})
		} else if strict && len(systemMessage) < 100 {
			issues = append(issues, ValidationIssue{
				Severity: "warning",
				Field:    "spec.declarative.systemMessage",
				Message:  "System message seems short. Consider providing more detailed instructions for the agent.",
			})
		}
	}

	// Check description
	if strict {
		description, _, _ := unstructured.NestedString(obj.Object, "spec", "description")
		if description == "" {
			issues = append(issues, ValidationIssue{
				Severity: "warning",
				Field:    "spec.description",
				Message:  "Consider adding a description to help users understand the agent's purpose",
			})
		}
	}

	return issues
}

func (ts *ToolServer) validateModelConfig(ctx context.Context, obj *unstructured.Unstructured, strict bool) []ValidationIssue {
	var issues []ValidationIssue

	// Check provider
	provider, found, _ := unstructured.NestedString(obj.Object, "spec", "provider")
	if !found || provider == "" {
		issues = append(issues, ValidationIssue{
			Severity: "error",
			Field:    "spec.provider",
			Message:  "spec.provider is required",
		})
	} else {
		validProviders := map[string]bool{
			"OpenAI": true, "AzureOpenAI": true, "Anthropic": true,
			"Gemini": true, "Ollama": true, "Custom": true,
		}
		if !validProviders[provider] {
			issues = append(issues, ValidationIssue{
				Severity: "error",
				Field:    "spec.provider",
				Message:  fmt.Sprintf("Invalid provider '%s'. Must be one of: OpenAI, AzureOpenAI, Anthropic, Gemini, Ollama, Custom", provider),
			})
		}
	}

	// Check model
	model, found, _ := unstructured.NestedString(obj.Object, "spec", "model")
	if !found || model == "" {
		issues = append(issues, ValidationIssue{
			Severity: "error",
			Field:    "spec.model",
			Message:  "spec.model is required",
		})
	}

	// Check apiKeySecret
	apiKeySecret, found, _ := unstructured.NestedString(obj.Object, "spec", "apiKeySecret")
	if !found || apiKeySecret == "" {
		// Only required for non-Ollama providers
		if provider != "Ollama" {
			issues = append(issues, ValidationIssue{
				Severity: "error",
				Field:    "spec.apiKeySecret",
				Message:  "spec.apiKeySecret is required for non-Ollama providers",
			})
		}
	}

	return issues
}

func (ts *ToolServer) validateMCPServer(ctx context.Context, obj *unstructured.Unstructured, strict bool) []ValidationIssue {
	var issues []ValidationIssue

	// Check deployment
	image, found, _ := unstructured.NestedString(obj.Object, "spec", "deployment", "image")
	if !found || image == "" {
		issues = append(issues, ValidationIssue{
			Severity: "error",
			Field:    "spec.deployment.image",
			Message:  "spec.deployment.image is required for MCPServer",
		})
	}

	// Check transportType
	transportType, _, _ := unstructured.NestedString(obj.Object, "spec", "transportType")
	if transportType != "" && transportType != "stdio" {
		issues = append(issues, ValidationIssue{
			Severity: "warning",
			Field:    "spec.transportType",
			Message:  fmt.Sprintf("Unexpected transportType '%s'. MCPServer typically uses 'stdio'", transportType),
		})
	}

	return issues
}

func (ts *ToolServer) validateRemoteMCPServer(ctx context.Context, obj *unstructured.Unstructured, strict bool) []ValidationIssue {
	var issues []ValidationIssue

	// Check URL
	url, found, _ := unstructured.NestedString(obj.Object, "spec", "url")
	if !found || url == "" {
		issues = append(issues, ValidationIssue{
			Severity: "error",
			Field:    "spec.url",
			Message:  "spec.url is required for RemoteMCPServer",
		})
	} else if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		issues = append(issues, ValidationIssue{
			Severity: "error",
			Field:    "spec.url",
			Message:  "spec.url must start with http:// or https://",
		})
	}

	// Check protocol
	protocol, _, _ := unstructured.NestedString(obj.Object, "spec", "protocol")
	if protocol != "" && protocol != "STREAMABLE_HTTP" && protocol != "SSE" {
		issues = append(issues, ValidationIssue{
			Severity: "error",
			Field:    "spec.protocol",
			Message:  "spec.protocol must be 'STREAMABLE_HTTP' or 'SSE'",
		})
	}

	return issues
}

// registerDiffManifest registers the diff_manifest tool.
func (ts *ToolServer) registerDiffManifest() {
	tool := mcp.NewTool("diff_manifest",
		mcp.WithDescription("Show the differences between a manifest and the current cluster state. Helps review changes before applying."),
		mcp.WithString("manifest",
			mcp.Required(),
			mcp.Description("YAML manifest to compare against current state"),
		),
	)

	ts.server.AddTool(tool, ts.handleDiffManifest)
}

func (ts *ToolServer) handleDiffManifest(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	manifest, _ := req.Params.Arguments["manifest"].(string)
	if manifest == "" {
		return mcp.NewToolResultError("manifest is required"), nil
	}

	// Parse manifest
	var obj unstructured.Unstructured
	if err := yaml.Unmarshal([]byte(manifest), &obj.Object); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse manifest: %v", err)), nil
	}

	name := obj.GetName()
	kind := obj.GetKind()

	// Try to get current state
	currentYAML, err := ts.k8sClient.GetCurrentState(ctx, kind, name)
	if err != nil {
		// Resource doesn't exist
		return mcp.NewToolResultText(fmt.Sprintf(`# New Resource

%s '%s' does not exist in the cluster.
This will CREATE a new resource.

Proposed manifest:
---
%s`, kind, name, manifest)), nil
	}

	// Parse current state for comparison
	var currentObj map[string]interface{}
	if err := yaml.Unmarshal([]byte(currentYAML), &currentObj); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse current state: %v", err)), nil
	}

	// Clean the proposed manifest for comparison
	proposedClean := make(map[string]interface{})
	for k, v := range obj.Object {
		if k != "status" {
			proposedClean[k] = v
		}
	}

	// Generate diff
	diff := cmp.Diff(currentObj, proposedClean)

	if diff == "" {
		return mcp.NewToolResultText(fmt.Sprintf("No changes detected. %s '%s' is already up to date.", kind, name)), nil
	}

	result := fmt.Sprintf(`# Diff: %s '%s'

Changes that will be applied:

%s

Legend: - removed, + added`, kind, name, diff)

	return mcp.NewToolResultText(result), nil
}

// registerApplyManifest registers the apply_manifest tool.
func (ts *ToolServer) registerApplyManifest() {
	tool := mcp.NewTool("apply_manifest",
		mcp.WithDescription("Apply a validated manifest to the Kubernetes cluster. IMPORTANT: Always validate and show diff to user before applying. Use dry_run=true to preview without applying."),
		mcp.WithString("manifest",
			mcp.Required(),
			mcp.Description("YAML manifest to apply"),
		),
		mcp.WithBoolean("dry_run",
			mcp.Description("Perform a server-side dry-run without actually applying (default: false)"),
		),
	)

	ts.server.AddTool(tool, ts.handleApplyManifest)
}

func (ts *ToolServer) handleApplyManifest(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	manifest, _ := req.Params.Arguments["manifest"].(string)
	if manifest == "" {
		return mcp.NewToolResultError("manifest is required"), nil
	}

	dryRun := false
	if v, ok := req.Params.Arguments["dry_run"].(bool); ok {
		dryRun = v
	}

	result, err := ts.k8sClient.Apply(ctx, manifest, dryRun)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to apply manifest: %v", err)), nil
	}

	var status string
	if dryRun {
		status = fmt.Sprintf("# Dry Run Successful\n\n%s '%s' in namespace '%s' would be %s.\n\nTo actually apply, run apply_manifest with dry_run=false.",
			result.Kind, result.Name, result.Namespace, result.Action)
	} else {
		status = fmt.Sprintf("# Successfully Applied\n\n%s '%s' in namespace '%s' has been %s.",
			result.Kind, result.Name, result.Namespace, result.Action)
	}

	return mcp.NewToolResultText(status), nil
}
