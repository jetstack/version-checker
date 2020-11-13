package architecture

import (
	"errors"
	"fmt"
	"sync"

	corev1 "k8s.io/api/core/v1"

	"github.com/jetstack/version-checker/pkg/api"
)

// NodeMetadata metadata about a particular node
type nodeMetadata struct {
	OS           api.OS
	Architecture api.Architecture
}

type NodeMap struct {
	mu    sync.RWMutex
	nodes map[string]*nodeMetadata
}

func New() *NodeMap {
	// might need to pass an initial map
	return &NodeMap{
		nodes: make(map[string]*nodeMetadata),
	}
}

func (m *NodeMap) GetArchitecture(nodeName string) (*nodeMetadata, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	meta, ok := m.nodes[nodeName]
	return meta, ok
}

func (m *NodeMap) Add(node *corev1.Node) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if node == nil {
		return errors.New("passed node is nil")
	}

	arch, ok := node.Labels[corev1.LabelArchStable]
	if !ok {
		return fmt.Errorf("missing %q label on node %q", corev1.LabelArchStable, node.Name)
	}

	os, ok := node.Labels[corev1.LabelOSStable]
	if !ok {
		return fmt.Errorf("missing %q label on node %q", corev1.LabelOSStable, node.Name)
	}

	m.nodes[node.Name] = &nodeMetadata{
		OS:           api.OS(os),
		Architecture: api.Architecture(arch),
	}
	return nil
}

func (m *NodeMap) Delete(nodeName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.nodes, nodeName)
}
