package localmodel

import (
	"context"
	"sync"
)

// Manager manages the Python model server
type Manager struct {
	server *PythonModelServer
	mu     sync.Mutex
}

// NewManager creates a new instance of the Python model server manager
func NewManager(server *PythonModelServer) *Manager {
	return &Manager{
		server: server,
	}
}

// Start starts the Python model server
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.server.Start(ctx)
}

// Stop stops the Python model server
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.server.Stop()
}

// GetServer returns the Python model server
func (m *Manager) GetServer() *PythonModelServer {
	return m.server
}
