// Package kubernetes provides a Kubernetes client wrapper for kagent resources.
package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"

	"github.com/kagent-dev/meta-kagent/pkg/types"
)

// Client wraps the Kubernetes dynamic client for kagent resources.
type Client struct {
	dynamicClient dynamic.Interface
	namespace     string
}

// GroupVersionResource definitions for kagent CRDs.
var (
	AgentGVR = schema.GroupVersionResource{
		Group:    "kagent.dev",
		Version:  "v1alpha2",
		Resource: "agents",
	}

	ModelConfigGVR = schema.GroupVersionResource{
		Group:    "kagent.dev",
		Version:  "v1alpha2",
		Resource: "modelconfigs",
	}

	MCPServerGVR = schema.GroupVersionResource{
		Group:    "kagent.dev",
		Version:  "v1alpha1",
		Resource: "mcpservers",
	}

	RemoteMCPServerGVR = schema.GroupVersionResource{
		Group:    "kagent.dev",
		Version:  "v1alpha2",
		Resource: "remotemcpservers",
	}
)

// NewClient creates a new Kubernetes client.
// It tries in-cluster config first, then falls back to kubeconfig.
func NewClient(namespace string) (*Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		configOverrides := &clientcmd.ConfigOverrides{}
		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
		config, err = kubeConfig.ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create kubernetes config: %w", err)
		}
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return &Client{
		dynamicClient: dynamicClient,
		namespace:     namespace,
	}, nil
}

// ListAgents lists all agents in the configured namespace.
func (c *Client) ListAgents(ctx context.Context) ([]types.Agent, error) {
	list, err := c.dynamicClient.Resource(AgentGVR).Namespace(c.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}

	var agents []types.Agent
	for _, item := range list.Items {
		agent, err := unstructuredToAgent(&item)
		if err != nil {
			return nil, err
		}
		agents = append(agents, *agent)
	}
	return agents, nil
}

// GetAgent gets a specific agent by name.
func (c *Client) GetAgent(ctx context.Context, name string) (*types.Agent, error) {
	obj, err := c.dynamicClient.Resource(AgentGVR).Namespace(c.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get agent %s: %w", name, err)
	}
	return unstructuredToAgent(obj)
}

// ListModelConfigs lists all model configs in the configured namespace.
func (c *Client) ListModelConfigs(ctx context.Context) ([]types.ModelConfig, error) {
	list, err := c.dynamicClient.Resource(ModelConfigGVR).Namespace(c.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list model configs: %w", err)
	}

	var configs []types.ModelConfig
	for _, item := range list.Items {
		config, err := unstructuredToModelConfig(&item)
		if err != nil {
			return nil, err
		}
		configs = append(configs, *config)
	}
	return configs, nil
}

// GetModelConfig gets a specific model config by name.
func (c *Client) GetModelConfig(ctx context.Context, name string) (*types.ModelConfig, error) {
	obj, err := c.dynamicClient.Resource(ModelConfigGVR).Namespace(c.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get model config %s: %w", name, err)
	}
	return unstructuredToModelConfig(obj)
}

// ListMCPServers lists all MCPServers in the configured namespace.
func (c *Client) ListMCPServers(ctx context.Context) ([]types.MCPServer, error) {
	list, err := c.dynamicClient.Resource(MCPServerGVR).Namespace(c.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list mcp servers: %w", err)
	}

	var servers []types.MCPServer
	for _, item := range list.Items {
		server, err := unstructuredToMCPServer(&item)
		if err != nil {
			return nil, err
		}
		servers = append(servers, *server)
	}
	return servers, nil
}

// ListRemoteMCPServers lists all RemoteMCPServers in the configured namespace.
func (c *Client) ListRemoteMCPServers(ctx context.Context) ([]types.RemoteMCPServer, error) {
	list, err := c.dynamicClient.Resource(RemoteMCPServerGVR).Namespace(c.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list remote mcp servers: %w", err)
	}

	var servers []types.RemoteMCPServer
	for _, item := range list.Items {
		server, err := unstructuredToRemoteMCPServer(&item)
		if err != nil {
			return nil, err
		}
		servers = append(servers, *server)
	}
	return servers, nil
}

// Apply applies a manifest (YAML string) to the cluster.
func (c *Client) Apply(ctx context.Context, manifest string, dryRun bool) (*ApplyResult, error) {
	// Parse the manifest
	var obj unstructured.Unstructured
	if err := yaml.Unmarshal([]byte(manifest), &obj.Object); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	gvr, err := gvrFromObject(&obj)
	if err != nil {
		return nil, err
	}

	// Set namespace if not specified
	if obj.GetNamespace() == "" {
		obj.SetNamespace(c.namespace)
	}

	opts := metav1.CreateOptions{}
	if dryRun {
		opts.DryRun = []string{metav1.DryRunAll}
	}

	// Try to get existing resource
	existing, err := c.dynamicClient.Resource(gvr).Namespace(obj.GetNamespace()).Get(ctx, obj.GetName(), metav1.GetOptions{})
	if err == nil {
		// Resource exists, update it
		obj.SetResourceVersion(existing.GetResourceVersion())
		updateOpts := metav1.UpdateOptions{}
		if dryRun {
			updateOpts.DryRun = []string{metav1.DryRunAll}
		}
		_, err = c.dynamicClient.Resource(gvr).Namespace(obj.GetNamespace()).Update(ctx, &obj, updateOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to update resource: %w", err)
		}
		return &ApplyResult{
			Action:    "updated",
			Kind:      obj.GetKind(),
			Name:      obj.GetName(),
			Namespace: obj.GetNamespace(),
			DryRun:    dryRun,
		}, nil
	}

	// Resource doesn't exist, create it
	_, err = c.dynamicClient.Resource(gvr).Namespace(obj.GetNamespace()).Create(ctx, &obj, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	return &ApplyResult{
		Action:    "created",
		Kind:      obj.GetKind(),
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
		DryRun:    dryRun,
	}, nil
}

// Delete deletes a resource from the cluster.
func (c *Client) Delete(ctx context.Context, kind, name string, dryRun bool) error {
	gvr, err := gvrFromKind(kind)
	if err != nil {
		return err
	}

	opts := metav1.DeleteOptions{}
	if dryRun {
		opts.DryRun = []string{metav1.DryRunAll}
	}

	return c.dynamicClient.Resource(gvr).Namespace(c.namespace).Delete(ctx, name, opts)
}

// GetCurrentState gets the current state of a resource for diffing.
func (c *Client) GetCurrentState(ctx context.Context, kind, name string) (string, error) {
	gvr, err := gvrFromKind(kind)
	if err != nil {
		return "", err
	}

	obj, err := c.dynamicClient.Resource(gvr).Namespace(c.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	// Remove server-managed fields for cleaner diff
	delete(obj.Object, "status")
	unstructured.RemoveNestedField(obj.Object, "metadata", "creationTimestamp")
	unstructured.RemoveNestedField(obj.Object, "metadata", "generation")
	unstructured.RemoveNestedField(obj.Object, "metadata", "resourceVersion")
	unstructured.RemoveNestedField(obj.Object, "metadata", "uid")
	unstructured.RemoveNestedField(obj.Object, "metadata", "managedFields")

	yamlBytes, err := yaml.Marshal(obj.Object)
	if err != nil {
		return "", fmt.Errorf("failed to marshal to yaml: %w", err)
	}

	return string(yamlBytes), nil
}

// ApplyResult contains the result of an apply operation.
type ApplyResult struct {
	Action    string `json:"action"`    // "created" or "updated"
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	DryRun    bool   `json:"dryRun"`
}

// Helper functions

func unstructuredToAgent(obj *unstructured.Unstructured) (*types.Agent, error) {
	data, err := json.Marshal(obj.Object)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal unstructured: %w", err)
	}
	var agent types.Agent
	if err := json.Unmarshal(data, &agent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to agent: %w", err)
	}
	return &agent, nil
}

func unstructuredToModelConfig(obj *unstructured.Unstructured) (*types.ModelConfig, error) {
	data, err := json.Marshal(obj.Object)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal unstructured: %w", err)
	}
	var config types.ModelConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to model config: %w", err)
	}
	return &config, nil
}

func unstructuredToMCPServer(obj *unstructured.Unstructured) (*types.MCPServer, error) {
	data, err := json.Marshal(obj.Object)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal unstructured: %w", err)
	}
	var server types.MCPServer
	if err := json.Unmarshal(data, &server); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to mcp server: %w", err)
	}
	return &server, nil
}

func unstructuredToRemoteMCPServer(obj *unstructured.Unstructured) (*types.RemoteMCPServer, error) {
	data, err := json.Marshal(obj.Object)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal unstructured: %w", err)
	}
	var server types.RemoteMCPServer
	if err := json.Unmarshal(data, &server); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to remote mcp server: %w", err)
	}
	return &server, nil
}

func gvrFromObject(obj *unstructured.Unstructured) (schema.GroupVersionResource, error) {
	return gvrFromKind(obj.GetKind())
}

func gvrFromKind(kind string) (schema.GroupVersionResource, error) {
	switch kind {
	case "Agent":
		return AgentGVR, nil
	case "ModelConfig":
		return ModelConfigGVR, nil
	case "MCPServer":
		return MCPServerGVR, nil
	case "RemoteMCPServer":
		return RemoteMCPServerGVR, nil
	default:
		return schema.GroupVersionResource{}, fmt.Errorf("unknown kind: %s", kind)
	}
}
