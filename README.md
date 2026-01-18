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
- OpenAI API key (or another supported LLM provider)

### Deploy KMeta-Agent

1. **Create the API key secret:**

```bash
kubectl create secret generic kmeta-agent-llm-secret \
  --namespace kagent \
  --from-literal=OPENAI_API_KEY=your-api-key-here
```

2. **Deploy the manifests:**

```bash
kubectl apply -k deploy/kubernetes/
```

3. **Verify the deployment:**

```bash
kubectl get agents,modelconfigs,mcpservers -n kagent -l app.kubernetes.io/part-of=kmeta-agent
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
make docker-build
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
├── deploy/kubernetes/       # K8s manifests
├── Dockerfile
├── Makefile
└── README.md
```

### Running Locally

```bash
# Requires kubeconfig access to a cluster with kagent
make run
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `KAGENT_NAMESPACE` | Namespace to manage | `kagent` |
| `LOG_LEVEL` | Log level (debug, info, warn, error) | `info` |

### Using a Different LLM Provider

Edit `deploy/kubernetes/modelconfig.yaml` to use a different provider:

```yaml
apiVersion: kagent.dev/v1alpha2
kind: ModelConfig
metadata:
  name: kmeta-agent-model
  namespace: kagent
spec:
  provider: Anthropic  # or Gemini, AzureOpenAI, Ollama
  model: claude-sonnet-4-20250514
  apiKeySecret: kmeta-agent-llm-secret
  apiKeySecretKey: ANTHROPIC_API_KEY
  anthropic: {}
```

## License

Apache 2.0
