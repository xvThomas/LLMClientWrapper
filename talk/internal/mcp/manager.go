package mcp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xvThomas/talk-backend/talk-libs/version"

	"github.com/xvThomas/talk-backend/talk/internal/domain"
)

const connectTimeout = 15 * time.Second

// ServerStatus holds the runtime state of a connected MCP server.
type ServerStatus struct {
	Config        ServerConfig
	Connected     bool
	ServerName    string // from Initialize response
	ServerVersion string // from Initialize response
	Tools         []string
	Error         string // non-empty if connection failed
}

// Manager manages connections to registered MCP servers.
type Manager struct {
	registry Registry
	mu       sync.RWMutex
	dial     func(context.Context, ServerConfig) (*mcp.ClientSession, error)
	sessions map[string]*mcp.ClientSession // keyed by ServerConfig.ID
	statuses []ServerStatus
	tools    []domain.Tool
}

// ErrSessionUnavailable indicates the MCP session could not be restored.
var ErrSessionUnavailable = errors.New("mcp session unavailable")

// NewManager creates a new Manager using the given Registry.
func NewManager(registry Registry) *Manager {
	return &Manager{
		registry: registry,
		sessions: make(map[string]*mcp.ClientSession),
	}
}

// ConnectAll connects to all registered MCP servers.
// Connection errors are recorded per server but do not abort the process.
func (m *Manager) ConnectAll(ctx context.Context) {
	configs, err := m.registry.List(ctx)
	if err != nil {
		m.mu.Lock()
		m.statuses = []ServerStatus{{Error: fmt.Sprintf("failed to list servers: %v", err)}}
		m.mu.Unlock()
		return
	}

	m.mu.Lock()
	m.tools = []domain.Tool{}
	m.statuses = []ServerStatus{}
	m.sessions = make(map[string]*mcp.ClientSession)
	m.mu.Unlock()

	for _, cfg := range configs {
		status, session, tools := m.connectServer(ctx, cfg)
		m.storeConnectionState(cfg, status, session, tools)
	}
}

// Connect connects to a single MCP server by config. Used for testing a new server on add.
func (m *Manager) Connect(ctx context.Context, cfg ServerConfig) (*ServerStatus, error) {
	status, session, tools := m.connectServer(ctx, cfg)
	if status.Error != "" {
		return &status, fmt.Errorf("%s", status.Error)
	}

	m.storeConnectionState(cfg, status, session, tools)
	return &status, nil
}

// EnsureConnected returns an active session for the given server, reconnecting it if needed.
func (m *Manager) EnsureConnected(ctx context.Context, id string) (*mcp.ClientSession, error) {
	cfg, err := m.registry.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: loading config for server %q: %v", ErrSessionUnavailable, id, err)
	}

	m.mu.RLock()
	session := m.sessions[id]
	m.mu.RUnlock()
	if session != nil {
		return session, nil
	}

	return m.reconnect(ctx, cfg)
}

// Reconnect recreates the MCP session for the given server ID.
func (m *Manager) Reconnect(ctx context.Context, id string) (*mcp.ClientSession, error) {
	cfg, err := m.registry.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: loading config for server %q: %v", ErrSessionUnavailable, id, err)
	}

	return m.reconnect(ctx, cfg)
}

func (m *Manager) reconnect(ctx context.Context, cfg ServerConfig) (*mcp.ClientSession, error) {
	slog.Info("attempting MCP reconnect",
		slog.String("server_id", cfg.ID),
		slog.String("server_name", cfg.Name),
		slog.String("server_url", cfg.URL),
	)

	status, session, tools := m.connectServer(ctx, cfg)
	if status.Error != "" {
		slog.Error("MCP reconnect failed",
			slog.String("server_id", cfg.ID),
			slog.String("server_name", cfg.Name),
			slog.String("server_url", cfg.URL),
			slog.String("error", status.Error),
		)
		m.storeConnectionState(cfg, status, nil, nil)
		return nil, fmt.Errorf("%w: %s", ErrSessionUnavailable, status.Error)
	}

	m.storeConnectionState(cfg, status, session, tools)
	slog.Info("MCP reconnect succeeded",
		slog.String("server_id", cfg.ID),
		slog.String("server_name", cfg.Name),
		slog.String("server_url", cfg.URL),
	)
	return session, nil
}

func (m *Manager) connectServer(ctx context.Context, cfg ServerConfig) (ServerStatus, *mcp.ClientSession, []domain.Tool) {
	status := ServerStatus{Config: cfg}
	session, err := m.connect(ctx, cfg)
	if err != nil {
		status.Error = err.Error()
		return status, nil, nil
	}

	status.Connected = true
	if res := session.InitializeResult(); res != nil && res.ServerInfo != nil {
		status.ServerName = res.ServerInfo.Name
		status.ServerVersion = res.ServerInfo.Version
	}

	toolsResult, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		status.Error = fmt.Sprintf("connected but failed to list tools: %v", err)
		_ = session.Close()
		return status, nil, nil
	}

	tools := make([]domain.Tool, 0, len(toolsResult.Tools))
	for _, t := range toolsResult.Tools {
		status.Tools = append(status.Tools, t.Name)
		tools = append(tools, &mcpToolAdapter{
			manager:    m,
			serverID:   cfg.ID,
			serverName: cfg.Name,
			tool:       *t,
		})
	}

	return status, session, tools
}

// Disconnect closes and removes a specific server connection and its tools.
func (m *Manager) Disconnect(id string) {
	m.mu.Lock()
	if session, ok := m.sessions[id]; ok {
		_ = session.Close()
		delete(m.sessions, id)
	}
	m.mu.Unlock()
	m.rebuildToolsExcluding(id)
}

// Refresh re-queries the tool list for all connected servers and rebuilds the
// internal tools slice. Returns the number of tools discovered.
func (m *Manager) Refresh(ctx context.Context) int {
	m.mu.Lock()
	statuses := append([]ServerStatus(nil), m.statuses...)
	sessions := make(map[string]*mcp.ClientSession, len(m.sessions))
	for id, session := range m.sessions {
		sessions[id] = session
	}
	m.mu.Unlock()

	refreshedTools := []domain.Tool{}
	for i := range statuses {
		st := &statuses[i]
		st.Tools = nil

		session, ok := sessions[st.Config.ID]
		if !ok || !st.Connected {
			continue
		}

		toolsResult, err := session.ListTools(ctx, &mcp.ListToolsParams{})
		if err != nil {
			st.Error = fmt.Sprintf("failed to refresh tools: %v", err)
			continue
		}

		st.Error = ""
		for _, t := range toolsResult.Tools {
			st.Tools = append(st.Tools, t.Name)
			refreshedTools = append(refreshedTools, &mcpToolAdapter{
				manager:    m,
				serverID:   st.Config.ID,
				serverName: st.Config.Name,
				tool:       *t,
			})
		}
	}

	m.mu.Lock()
	m.tools = refreshedTools
	m.statuses = statuses
	count := len(m.tools)
	m.mu.Unlock()
	return count
}

func (m *Manager) rebuildToolsExcluding(excludeID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var filtered []domain.Tool
	for _, t := range m.tools {
		if adapter, ok := t.(*mcpToolAdapter); ok {
			for _, st := range m.statuses {
				if st.Config.ID == excludeID && adapter.serverID == st.Config.ID {
					goto skip
				}
			}
		}
		filtered = append(filtered, t)
	skip:
	}
	m.tools = filtered

	var filteredStatus []ServerStatus
	for _, st := range m.statuses {
		if st.Config.ID != excludeID {
			filteredStatus = append(filteredStatus, st)
		}
	}
	m.statuses = filteredStatus
}

// Tools returns all tools from all connected MCP servers.
func (m *Manager) Tools() []domain.Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tools := make([]domain.Tool, len(m.tools))
	copy(tools, m.tools)
	return tools
}

// Statuses returns the connection status for all servers.
func (m *Manager) Statuses() []ServerStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	statuses := make([]ServerStatus, len(m.statuses))
	copy(statuses, m.statuses)
	return statuses
}

// Close closes all active MCP sessions.
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, session := range m.sessions {
		_ = session.Close()
		delete(m.sessions, id)
	}
}

func (m *Manager) storeConnectionState(cfg ServerConfig, status ServerStatus, session *mcp.ClientSession, tools []domain.Tool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.sessions[cfg.ID]; ok && existing != nil && existing != session {
		_ = existing.Close()
	}

	if session == nil {
		delete(m.sessions, cfg.ID)
	} else {
		m.sessions[cfg.ID] = session
	}

	statusIndex := -1
	for i := range m.statuses {
		if m.statuses[i].Config.ID == cfg.ID {
			statusIndex = i
			break
		}
	}
	if statusIndex >= 0 {
		m.statuses[statusIndex] = status
	} else {
		m.statuses = append(m.statuses, status)
	}

	filteredTools := m.tools[:0]
	for _, tool := range m.tools {
		adapter, ok := tool.(*mcpToolAdapter)
		if ok && adapter.serverID == cfg.ID {
			continue
		}
		filteredTools = append(filteredTools, tool)
	}
	m.tools = append(filteredTools, tools...)
}

func isReconnectableCallError(err error) bool {
	if err == nil {
		return false
	}

	message := strings.ToLower(err.Error())
	reconnectableMarkers := []string{
		"broken pipe",
		"connection closed",
		"connection reset",
		"connection refused",
		"client is closing",
		"eof",
		"closed network connection",
		"use of closed network connection",
		"stream closed",
		"transport is closing",
		"session closed",
	}

	for _, marker := range reconnectableMarkers {
		if strings.Contains(message, marker) {
			return true
		}
	}

	return false
}

func (m *Manager) connect(ctx context.Context, cfg ServerConfig) (*mcp.ClientSession, error) {
	if m.dial != nil {
		return m.dial(ctx, cfg)
	}

	connectCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	httpClient := buildHTTPClient(cfg)
	transport := &mcp.StreamableClientTransport{
		Endpoint:   cfg.URL,
		HTTPClient: httpClient,
	}
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "talk-cli",
		Version: version.Version,
	}, nil)

	session, err := client.Connect(connectCtx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("connecting to %q (%s): %w", cfg.Name, cfg.URL, err)
	}
	return session, nil
}

// buildHTTPClient returns an *http.Client configured for the server's auth type.
func buildHTTPClient(cfg ServerConfig) *http.Client {
	switch cfg.AuthType {
	case AuthTypeAPIKey:
		if cfg.APIKey == "" {
			return http.DefaultClient
		}
		return &http.Client{
			Transport: &apiKeyTransport{
				key:  cfg.APIKey,
				base: http.DefaultTransport,
			},
		}
	default:
		return http.DefaultClient
	}
}

// apiKeyTransport injects an X-API-Key header into every request.
type apiKeyTransport struct {
	key  string
	base http.RoundTripper
}

func (t *apiKeyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	r.Header.Set("X-API-Key", t.key)
	return t.base.RoundTrip(r)
}
