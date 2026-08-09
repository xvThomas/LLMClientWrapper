package mcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	projectlogger "github.com/xvThomas/talk-backend/talk-libs/logger"
	"github.com/xvThomas/talk-backend/talk-libs/version"

	"github.com/xvThomas/talk-backend/talk/internal/domain"
)

const (
	connectTimeout     = 15 * time.Second
	logMsgMCPReconnect = "mcp reconnect"
)

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
	registry       Registry
	mu             sync.RWMutex
	reconnectGroup singleflight.Group
	reconnectSeq   atomic.Uint64
	log            *slog.Logger
	dial           func(context.Context, ServerConfig) (*mcp.ClientSession, error)
	sessions       map[string]*mcp.ClientSession // keyed by ServerConfig.ID
	statuses       []ServerStatus
	tools          []domain.Tool
}

// ErrSessionUnavailable indicates the MCP session could not be restored.
var ErrSessionUnavailable = errors.New("mcp session unavailable")

// NewManager creates a new Manager using the given Registry.
func NewManager(registry Registry) *Manager {
	log := projectlogger.Logger
	if log == nil {
		log = slog.Default()
	}

	return &Manager{
		registry: registry,
		log:      log,
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
		status, session, tools, _ := m.connectServer(ctx, cfg)
		m.storeConnectionState(cfg, status, session, tools)
	}
}

// Connect connects to a single MCP server by config. Used for testing a new server on add.
func (m *Manager) Connect(ctx context.Context, cfg ServerConfig) (*ServerStatus, error) {
	status, session, tools, err := m.connectServer(ctx, cfg)
	if err != nil {
		return &status, err
	}

	m.storeConnectionState(cfg, status, session, tools)
	return &status, nil
}

// EnsureConnected returns an active session for the given server, reconnecting it if needed.
func (m *Manager) EnsureConnected(ctx context.Context, id string) (*mcp.ClientSession, error) {
	cfg, err := m.registry.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: loading config for server %q: %w", ErrSessionUnavailable, id, err)
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
		return nil, fmt.Errorf("%w: loading config for server %q: %w", ErrSessionUnavailable, id, err)
	}

	return m.reconnect(ctx, cfg)
}

func (m *Manager) reconnect(ctx context.Context, cfg ServerConfig) (*mcp.ClientSession, error) {
	v, err, _ := m.reconnectGroup.Do(cfg.ID, func() (any, error) {
		reconnectID := m.nextReconnectID(cfg.ID)
		startedAt := time.Now()

		m.log.Info(logMsgMCPReconnect,
			"reconnect_event", "attempt",
			"outcome", "attempt",
			"correlation_id", reconnectID,
			"server_id", cfg.ID,
			"server_name", cfg.Name,
			"server_url", cfg.URL,
		)

		status, session, tools, _ := m.connectServer(ctx, cfg)
		if status.Error != "" {
			return nil, m.applyReconnectFailure(reconnectID, startedAt, cfg, status)
		}

		m.storeConnectionState(cfg, status, session, tools)
		m.log.Info(logMsgMCPReconnect,
			"reconnect_event", "result",
			"outcome", "success",
			"correlation_id", reconnectID,
			"duration_ms", time.Since(startedAt).Milliseconds(),
			"tool_count", len(tools),
			"server_id", cfg.ID,
			"server_name", cfg.Name,
			"server_url", cfg.URL,
		)
		return session, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*mcp.ClientSession), nil
}

// applyReconnectFailure logs the reconnect failure, updates the stored status,
// and returns an ErrSessionUnavailable-wrapped error.
func (m *Manager) applyReconnectFailure(reconnectID string, startedAt time.Time, cfg ServerConfig, status ServerStatus) error {
	m.log.Error(logMsgMCPReconnect,
		"reconnect_event", "result",
		"outcome", "failure",
		"correlation_id", reconnectID,
		"duration_ms", time.Since(startedAt).Milliseconds(),
		"error_class", classifyReconnectError(status.Error),
		"server_id", cfg.ID,
		"server_name", cfg.Name,
		"server_url", cfg.URL,
		"error", status.Error,
	)
	m.mu.Lock()
	for i := range m.statuses {
		if m.statuses[i].Config.ID == cfg.ID {
			m.statuses[i] = status
			break
		}
	}
	m.mu.Unlock()
	return fmt.Errorf("%w: %s", ErrSessionUnavailable, status.Error)
}

func (m *Manager) nextReconnectID(serverID string) string {
	seq := m.reconnectSeq.Add(1)
	return fmt.Sprintf("%s-%d", serverID, seq)
}

func classifyReconnectError(message string) string {
	lower := strings.ToLower(message)
	switch {
	case strings.Contains(lower, "timeout"):
		return "timeout"
	case strings.Contains(lower, "closed") || strings.Contains(lower, "broken pipe") || strings.Contains(lower, "reset"):
		return "network"
	case strings.Contains(lower, "auth") || strings.Contains(lower, "unauthorized") || strings.Contains(lower, "forbidden"):
		return "auth"
	default:
		return "unknown"
	}
}

func (m *Manager) connectServer(ctx context.Context, cfg ServerConfig) (ServerStatus, *mcp.ClientSession, []domain.Tool, error) {
	status := ServerStatus{Config: cfg}
	session, err := m.connect(ctx, cfg)
	if err != nil {
		status.Error = err.Error()
		return status, nil, nil, err
	}

	status.Connected = true
	if res := session.InitializeResult(); res != nil && res.ServerInfo != nil {
		status.ServerName = res.ServerInfo.Name
		status.ServerVersion = res.ServerInfo.Version
	}

	toolsResult, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		wrappedErr := fmt.Errorf("connected but failed to list tools: %w", err)
		status.Error = wrappedErr.Error()
		_ = session.Close()
		return status, nil, nil, wrappedErr
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

	return status, session, tools, nil
}

// Disconnect closes and removes a specific server connection and its tools.
func (m *Manager) Disconnect(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.sessions[id]; ok {
		_ = session.Close()
		delete(m.sessions, id)
	}

	var filtered []domain.Tool
	for _, t := range m.tools {
		if adapter, ok := t.(*mcpToolAdapter); ok && adapter.serverID == id {
			continue
		}
		filtered = append(filtered, t)
	}
	m.tools = filtered

	var filteredStatus []ServerStatus
	for _, st := range m.statuses {
		if st.Config.ID != id {
			filteredStatus = append(filteredStatus, st)
		}
	}
	m.statuses = filteredStatus
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
		session, ok := sessions[st.Config.ID]
		if !ok || !st.Connected {
			continue
		}
		refreshedTools = append(refreshedTools, m.refreshOneServer(ctx, st, session)...)
	}

	// Track which server IDs were in this refresh snapshot.
	refreshedIDs := make(map[string]bool, len(statuses))
	for _, st := range statuses {
		refreshedIDs[st.Config.ID] = true
	}

	m.mu.Lock()
	m.mergeRefreshedStatuses(statuses)
	m.tools = append(filterToolsNotIn(m.tools, refreshedIDs), refreshedTools...)
	count := len(m.tools)
	m.mu.Unlock()
	return count
}

// refreshOneServer re-queries the tool list for a single server and updates its
// status in place. Returns the freshly constructed tool adapters.
func (m *Manager) refreshOneServer(ctx context.Context, st *ServerStatus, session *mcp.ClientSession) []domain.Tool {
	st.Tools = nil
	toolsResult, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		st.Error = fmt.Sprintf("failed to refresh tools: %v", err)
		return nil
	}
	st.Error = ""
	tools := make([]domain.Tool, 0, len(toolsResult.Tools))
	for _, t := range toolsResult.Tools {
		st.Tools = append(st.Tools, t.Name)
		tools = append(tools, &mcpToolAdapter{
			manager:    m,
			serverID:   st.Config.ID,
			serverName: st.Config.Name,
			tool:       *t,
		})
	}
	return tools
}

// mergeRefreshedStatuses updates m.statuses in place with refreshed snapshots,
// appending any server that was added after the snapshot was taken.
// Caller must hold m.mu.
func (m *Manager) mergeRefreshedStatuses(statuses []ServerStatus) {
	for _, st := range statuses {
		found := false
		for i := range m.statuses {
			if m.statuses[i].Config.ID == st.Config.ID {
				m.statuses[i] = st
				found = true
				break
			}
		}
		if !found {
			m.statuses = append(m.statuses, st)
		}
	}
}

// filterToolsNotIn returns the tools whose server ID is not in serverIDs.
func filterToolsNotIn(tools []domain.Tool, serverIDs map[string]bool) []domain.Tool {
	var kept []domain.Tool
	for _, t := range tools {
		if adapter, ok := t.(*mcpToolAdapter); ok && serverIDs[adapter.serverID] {
			continue
		}
		kept = append(kept, t)
	}
	return kept
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

	var toClose *mcp.ClientSession
	if existing, ok := m.sessions[cfg.ID]; ok && existing != nil && existing != session {
		toClose = existing
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

	var filteredTools []domain.Tool
	for _, tool := range m.tools {
		adapter, ok := tool.(*mcpToolAdapter)
		if ok && adapter.serverID == cfg.ID {
			continue
		}
		filteredTools = append(filteredTools, tool)
	}
	// Clear old backing-array slots so displaced adapters can be GC'd.
	for i := range m.tools {
		m.tools[i] = nil
	}
	m.tools = append(filteredTools, tools...)

	m.mu.Unlock()

	// Close the displaced session outside the lock to avoid blocking under contention.
	if toClose != nil {
		_ = toClose.Close()
	}
}

func isReconnectableCallError(err error) bool {
	if err == nil {
		return false
	}
	// Prefer typed error checks — stable across SDK and OS versions.
	if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
		return true
	}
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return true
	}
	// Fall back to string matching for SDK-level errors not surfaced as typed values.
	message := strings.ToLower(err.Error())
	for _, marker := range []string{
		"broken pipe",
		"connection closed",
		"connection reset by peer",
		"client is closing",
		"closed network connection",
		"stream closed",
		"transport is closing",
		"session closed",
	} {
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
