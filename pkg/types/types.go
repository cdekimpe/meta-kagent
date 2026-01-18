// Package types defines the kagent CRD types used by the meta-agent.
package types

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Agent represents a kagent Agent resource.
type Agent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              AgentSpec   `json:"spec,omitempty"`
	Status            AgentStatus `json:"status,omitempty"`
}

// AgentSpec defines the desired state of an Agent.
type AgentSpec struct {
	Type        string           `json:"type,omitempty"` // "Declarative" or "BYO"
	Description string           `json:"description,omitempty"`
	Declarative *DeclarativeSpec `json:"declarative,omitempty"`
	A2AConfig   *A2AConfig       `json:"a2aConfig,omitempty"`
}

// DeclarativeSpec defines a declarative agent configuration.
type DeclarativeSpec struct {
	ModelConfig   string     `json:"modelConfig,omitempty"`
	SystemMessage string     `json:"systemMessage,omitempty"`
	Tools         []ToolSpec `json:"tools,omitempty"`
}

// ToolSpec defines a tool reference.
type ToolSpec struct {
	Type      string         `json:"type,omitempty"` // "McpServer"
	McpServer *McpServerRef  `json:"mcpServer,omitempty"`
}

// McpServerRef references an MCP server and its tools.
type McpServerRef struct {
	Name      string   `json:"name,omitempty"`
	Kind      string   `json:"kind,omitempty"` // "MCPServer", "RemoteMCPServer", "Service"
	APIGroup  string   `json:"apiGroup,omitempty"`
	ToolNames []string `json:"toolNames,omitempty"`
}

// A2AConfig defines agent-to-agent configuration.
type A2AConfig struct {
	Skills []Skill `json:"skills,omitempty"`
}

// Skill defines an agent skill for A2A communication.
type Skill struct {
	ID          string   `json:"id,omitempty"`
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	InputModes  []string `json:"inputModes,omitempty"`
	OutputModes []string `json:"outputModes,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Examples    []string `json:"examples,omitempty"`
}

// AgentStatus defines the observed state of an Agent.
type AgentStatus struct {
	Ready    bool   `json:"ready,omitempty"`
	Accepted bool   `json:"accepted,omitempty"`
	Message  string `json:"message,omitempty"`
}

// AgentList contains a list of Agents.
type AgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Agent `json:"items"`
}

// ModelConfig represents a kagent ModelConfig resource.
type ModelConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ModelConfigSpec `json:"spec,omitempty"`
}

// ModelConfigSpec defines the desired state of a ModelConfig.
type ModelConfigSpec struct {
	Provider        string                 `json:"provider,omitempty"` // "OpenAI", "AzureOpenAI", "Anthropic", "Gemini", "Ollama", "Custom"
	Model           string                 `json:"model,omitempty"`
	APIKeySecret    string                 `json:"apiKeySecret,omitempty"`
	APIKeySecretKey string                 `json:"apiKeySecretKey,omitempty"`
	BaseURL         string                 `json:"baseUrl,omitempty"`
	OpenAI          map[string]interface{} `json:"openai,omitempty"`
	Anthropic       map[string]interface{} `json:"anthropic,omitempty"`
	Gemini          map[string]interface{} `json:"gemini,omitempty"`
	Azure           map[string]interface{} `json:"azure,omitempty"`
	Ollama          map[string]interface{} `json:"ollama,omitempty"`
}

// ModelConfigList contains a list of ModelConfigs.
type ModelConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ModelConfig `json:"items"`
}

// MCPServer represents a kagent MCPServer resource (local stdio transport).
type MCPServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              MCPServerSpec `json:"spec,omitempty"`
}

// MCPServerSpec defines the desired state of an MCPServer.
type MCPServerSpec struct {
	Description    string          `json:"description,omitempty"`
	Deployment     *DeploymentSpec `json:"deployment,omitempty"`
	TransportType  string          `json:"transportType,omitempty"` // "stdio"
	StdioTransport map[string]interface{} `json:"stdioTransport,omitempty"`
}

// DeploymentSpec defines the container deployment for an MCPServer.
type DeploymentSpec struct {
	Image     string            `json:"image,omitempty"`
	Cmd       string            `json:"cmd,omitempty"`
	Args      []string          `json:"args,omitempty"`
	Port      int32             `json:"port,omitempty"`
	Env       []EnvVar          `json:"env,omitempty"`
	Resources *ResourceRequirements `json:"resources,omitempty"`
}

// EnvVar defines an environment variable.
type EnvVar struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

// ResourceRequirements defines resource requests and limits.
type ResourceRequirements struct {
	Requests map[string]string `json:"requests,omitempty"`
	Limits   map[string]string `json:"limits,omitempty"`
}

// MCPServerList contains a list of MCPServers.
type MCPServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MCPServer `json:"items"`
}

// RemoteMCPServer represents a kagent RemoteMCPServer resource (HTTP/SSE transport).
type RemoteMCPServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              RemoteMCPServerSpec `json:"spec,omitempty"`
}

// RemoteMCPServerSpec defines the desired state of a RemoteMCPServer.
type RemoteMCPServerSpec struct {
	Description      string `json:"description,omitempty"`
	URL              string `json:"url,omitempty"`
	Protocol         string `json:"protocol,omitempty"` // "STREAMABLE_HTTP" or "SSE"
	Timeout          string `json:"timeout,omitempty"`
	SSEReadTimeout   string `json:"sseReadTimeout,omitempty"`
	TerminateOnClose bool   `json:"terminateOnClose,omitempty"`
}

// RemoteMCPServerList contains a list of RemoteMCPServers.
type RemoteMCPServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RemoteMCPServer `json:"items"`
}
