# KMeta-Agent

**KMeta-Agent** is a human-in-the-loop agent architect for the [kagent](https://kagent.dev) platform. It helps you design, review, apply, and evolve kagent agents by producing proposed Kubernetes manifests.

## Overview

KMeta-Agent is itself a kagent agent that provides tools for managing kagent resources:

- **Agents**: Design, create, update, and delete agents
- **ModelConfigs**: Configure LLM providers (OpenAI, Anthropic, Gemini, etc.)
- **MCPServers**: Set up MCP tool servers (local or remote)
- **RBAC**: Generate appropriate permissions for agents

### Key Features

- **Human-in-the-Loop**: Never applies changes without explicit user approval
- **Validation**: Validates manifests before applying
- **Diff Preview**: Shows what will change before mutations
- **Best Practices**: Guides you through agent design with best practices

## Installation

### Prerequisites

- Kubernetes cluster with kagent installed
- `kubectl` configured to access your cluster
- `helm` v3+ installed
- Docker for building the image

KMeta-Agent uses your platform's existing ModelConfig (e.g., `default-model-config`), so no additional API keys are required.

### Quick Start

1. **Build and push the image to your container registry:**

```bash
# Set your container registry
export IMAGE_REPO=ghcr.io/yourusername

# Build and push
make docker-release
```

2. **Deploy with Helm:**

```bash
helm install kmeta-agent deploy/helm/kmeta-agent/ \
  --namespace kagent \
  --set mcpServer.image.repository=ghcr.io/yourusername/meta-kagent
```

3. **Verify the deployment:**

```bash
kubectl get agents,mcpservers -n kagent -l app.kubernetes.io/name=kmeta-agent
kubectl get pods -n kagent -l app.kubernetes.io/name=kmeta-agent-tools
```

## Usage

Once deployed, you can interact with KMeta-Agent through the kagent UI or CLI.

### Example Interactions

**Design a new agent:**
```
You: I want to create an agent that helps with Prometheus monitoring.

KMeta-Agent: I'd be happy to help you design a Prometheus monitoring agent.
Let me first check what MCP servers are available...
[uses list_mcp_servers]

Based on available tools, here's what I propose:
- Name: prometheus-assistant
- Purpose: Help users write PromQL queries and manage alerts
- Tools: prometheus_query, prometheus_get_alerts

Would you like me to generate the manifest?
```

**Review an existing agent:**
```
You: Review the k8s-troubleshooter agent

KMeta-Agent: Let me fetch the agent details...
[uses get_agent]

Here's my analysis:
- System prompt is well-structured with clear responsibilities
- Tool selection is appropriate for Kubernetes troubleshooting
- Suggestion: Consider adding the `k8s_get_events` tool for better diagnostics
```

**Apply changes:**
```
You: Apply that manifest

KMeta-Agent: Before applying, let me validate and show you the diff...
[uses validate_manifest]
[uses diff_manifest]

Validation passed. Here's what will change:
- CREATE Agent 'prometheus-assistant' in namespace 'kagent'

Do you want me to apply this? (yes/no)

You: yes

KMeta-Agent: Applied successfully! The prometheus-assistant agent is now created.
```

## Available Tools

| Tool | Description |
|------|-------------|
| `list_agents` | List all agents in the namespace |
| `get_agent` | Get detailed information about an agent |
| `create_agent_manifest` | Generate a new agent manifest |
| `update_agent_manifest` | Modify an existing agent |
| `delete_agent` | Delete an agent |
| `list_model_configs` | List available model configurations |
| `create_model_config_manifest` | Generate a model config manifest |
| `list_mcp_servers` | List MCP servers |
| `create_mcp_server_manifest` | Generate an MCP server manifest |
| `generate_rbac_manifest` | Generate RBAC manifests |
| `validate_manifest` | Validate a manifest |
| `diff_manifest` | Show diff against current state |
| `apply_manifest` | Apply a manifest to the cluster |

## Configuration

### Using a Different ModelConfig

KMeta-Agent uses the platform's existing ModelConfig. To use a different one:

```bash
# List available ModelConfigs
kubectl get modelconfigs -n kagent

# Install with a specific ModelConfig
helm install kmeta-agent deploy/helm/kmeta-agent/ \
  --namespace kagent \
  --set modelConfig.name=your-model-config \
  --set mcpServer.image.repository=ghcr.io/yourusername/meta-kagent
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `KAGENT_NAMESPACE` | Namespace to manage | `kagent` |
| `LOG_LEVEL` | Log level (debug, info, warn, error) | `info` |

## Development

### Building from Source

```bash
# Download dependencies
make deps

# Build the binary
make build

# Run tests
make test

# Build Docker image
make docker-build IMAGE_REPO=ghcr.io/yourusername
```

### Project Structure

```
meta-kagent/
├── cmd/mcp-server/          # Entry point
├── internal/
│   ├── kubernetes/          # K8s client wrapper
│   ├── server/              # MCP server
│   ├── tools/               # Tool implementations
│   └── validation/          # Manifest validation
├── pkg/types/               # kagent CRD types
├── deploy/
│   ├── helm/kmeta-agent/    # Helm chart (recommended)
│   └── kubernetes/          # Kustomize manifests (legacy)
├── Dockerfile
├── Makefile
└── README.md
```

### Running Locally

```bash
# Requires kubeconfig access to a cluster with kagent
make run
```

### Makefile Targets

```bash
make help              # Show all available targets
make docker-release    # Build and push image
make helm-install      # Install with Helm
make helm-uninstall    # Uninstall with Helm
make helm-upgrade      # Upgrade with Helm
```

## Uninstalling

```bash
# Using Helm
helm uninstall kmeta-agent -n kagent

# Or using Makefile
make helm-uninstall
```

## License

Apache 2.0
