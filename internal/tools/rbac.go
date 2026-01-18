package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// registerGenerateRBACManifest registers the generate_rbac_manifest tool.
func (ts *ToolServer) registerGenerateRBACManifest() {
	tool := mcp.NewTool("generate_rbac_manifest",
		mcp.WithDescription("Generate RBAC manifests (ServiceAccount, Role, RoleBinding) for a kagent agent. Provides appropriate permissions for the agent to access its required resources."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Base name for RBAC resources (e.g., 'my-agent' creates 'my-agent' ServiceAccount, 'my-agent-role' Role, etc.)"),
		),
		mcp.WithString("permissions",
			mcp.Description("Permission preset: 'readonly' (read kagent resources), 'standard' (read/write kagent resources), or 'admin' (full kagent + RBAC management). Default: 'readonly'"),
		),
		mcp.WithString("additional_rules_json",
			mcp.Description("JSON array of additional RBAC rules. Format: [{\"apiGroups\": [\"...\"], \"resources\": [\"...\"], \"verbs\": [\"...\"]}]"),
		),
	)

	ts.server.AddTool(tool, ts.handleGenerateRBACManifest)
}

func (ts *ToolServer) handleGenerateRBACManifest(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, _ := req.Params.Arguments["name"].(string)
	permissions, _ := req.Params.Arguments["permissions"].(string)

	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}

	if permissions == "" {
		permissions = "readonly"
	}

	validPermissions := map[string]bool{"readonly": true, "standard": true, "admin": true}
	if !validPermissions[permissions] {
		return mcp.NewToolResultError("permissions must be 'readonly', 'standard', or 'admin'"), nil
	}

	// Generate ServiceAccount
	serviceAccount := fmt.Sprintf(`apiVersion: v1
kind: ServiceAccount
metadata:
  name: %s
  namespace: kagent
  labels:
    app.kubernetes.io/name: %s
    app.kubernetes.io/component: rbac`, name, name)

	// Generate Role based on permission level
	var rules string
	switch permissions {
	case "readonly":
		rules = `  # Read-only access to kagent resources
  - apiGroups: ["kagent.dev"]
    resources: ["agents", "modelconfigs", "mcpservers", "remotemcpservers"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["kagent.dev"]
    resources: ["agents/status"]
    verbs: ["get", "list", "watch"]`

	case "standard":
		rules = `  # Read/write access to kagent resources
  - apiGroups: ["kagent.dev"]
    resources: ["agents", "modelconfigs", "mcpservers", "remotemcpservers"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  - apiGroups: ["kagent.dev"]
    resources: ["agents/status"]
    verbs: ["get", "list", "watch"]
  # Read secrets for validation
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get", "list"]`

	case "admin":
		rules = `  # Full access to kagent resources
  - apiGroups: ["kagent.dev"]
    resources: ["agents", "modelconfigs", "mcpservers", "remotemcpservers"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  - apiGroups: ["kagent.dev"]
    resources: ["agents/status"]
    verbs: ["get", "list", "watch"]
  # Read secrets for validation
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get", "list"]
  # Manage ServiceAccounts
  - apiGroups: [""]
    resources: ["serviceaccounts"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  # Manage RBAC within namespace
  - apiGroups: ["rbac.authorization.k8s.io"]
    resources: ["roles", "rolebindings"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]`
	}

	role := fmt.Sprintf(`apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: %s-role
  namespace: kagent
  labels:
    app.kubernetes.io/name: %s
    app.kubernetes.io/component: rbac
rules:
%s`, name, name, rules)

	// Generate RoleBinding
	roleBinding := fmt.Sprintf(`apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: %s-rolebinding
  namespace: kagent
  labels:
    app.kubernetes.io/name: %s
    app.kubernetes.io/component: rbac
subjects:
  - kind: ServiceAccount
    name: %s
    namespace: kagent
roleRef:
  kind: Role
  name: %s-role
  apiGroup: rbac.authorization.k8s.io`, name, name, name, name)

	result := fmt.Sprintf(`# Generated RBAC Manifests for '%s'
# Permission level: %s
# Review these manifests before applying.

---
%s
---
%s
---
%s
`, name, permissions, serviceAccount, role, roleBinding)

	// Add description of what each permission level provides
	var permissionDesc string
	switch permissions {
	case "readonly":
		permissionDesc = "This grants read-only access to kagent resources (agents, model configs, MCP servers)."
	case "standard":
		permissionDesc = "This grants read/write access to kagent resources and read access to secrets for validation."
	case "admin":
		permissionDesc = "This grants full access to kagent resources plus the ability to manage RBAC and ServiceAccounts."
	}

	result = strings.Replace(result, "# Review", fmt.Sprintf("# %s\n# Review", permissionDesc), 1)

	return mcp.NewToolResultText(result), nil
}
