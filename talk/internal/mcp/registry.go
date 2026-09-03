package mcp

import (
	"context"
	"fmt"
	"regexp"
)

// ToolNameSeparator joins a server name and a remote tool name into the
// namespaced tool name exposed to the LLM (e.g. "owm__geocode").
const ToolNameSeparator = "__"

// MaxServerNameLength bounds server names so namespaced tool names stay within
// the 64-character limit imposed by LLM providers on function names.
const MaxServerNameLength = 24

// serverNameRe keeps server names usable verbatim as a tool name prefix:
// lowercase alphanumerics and hyphens only. Underscores are excluded so that
// ToolNameSeparator can never appear inside the prefix itself.
var serverNameRe = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,22}[a-z0-9])?$`)

// ValidateServerName reports whether name can be used as a tool namespace prefix.
func ValidateServerName(name string) error {
	if !serverNameRe.MatchString(name) {
		return fmt.Errorf(
			"must be 1 to %d characters, lowercase letters, digits or hyphens, starting and ending with a letter or digit (no underscore)",
			MaxServerNameLength,
		)
	}
	return nil
}

// AuthType represents the authentication method for an MCP server.
type AuthType string

const (
	AuthTypeNone   AuthType = "none"
	AuthTypeAPIKey AuthType = "apikey"
	AuthTypeOAuth  AuthType = "oauth"
)

// OAuthConfig holds OAuth 2.0 credentials for an MCP server.
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
	Scopes       []string
}

// ServerConfig represents a registered MCP server.
type ServerConfig struct {
	ID       string
	Name     string
	URL      string
	AuthType AuthType
	APIKey   string       // populated when AuthType == AuthTypeAPIKey
	OAuth    *OAuthConfig // populated when AuthType == AuthTypeOAuth
}

// Registry provides CRUD operations for MCP server configurations.
type Registry interface {
	Add(ctx context.Context, cfg ServerConfig) error
	Remove(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (ServerConfig, error)
	List(ctx context.Context) ([]ServerConfig, error)
}
