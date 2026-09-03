package mcpserver

import (
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// DefaultHTTPPort is the port used for the HTTP transport when HTTP_PORT is unset.
const DefaultHTTPPort = 8080

// DefaultHTTPHost is the interface the HTTP transport binds to when HTTP_HOST is unset.
const DefaultHTTPHost = "localhost"

// BaseEnv holds the environment configuration common to all MCP servers.
type BaseEnv struct {
	APIKey                   string // X_API_KEY — shared secret for authenticating HTTP clients
	BaseURL                  string // BASE_URL — public-facing URL of this server (required for Claude Desktop)
	OAuthAuthorizationServer string // OAUTH_AUTHORIZATION_SERVER — issuer URL of the external AS
	OAuthAudience            string // OAUTH_AUDIENCE — expected aud claim in the JWT (API identifier)
	OAuthScopes              string // OAUTH_SCOPES — comma-separated list of required scopes
	OAuthClientSecret        string // OAUTH_CLIENT_SECRET — client secret injected by the AS proxy token endpoint (confidential clients only)

	// HTTP server settings.
	HTTPHost string // HTTP_HOST — interface to bind to for the HTTP transport (default: localhost)
	HTTPPort int    // HTTP_PORT — port to listen on for the HTTP transport (default: 8080)

	// HTTP security settings.
	HTTPRateLimit    int           // HTTP_RATE_LIMIT — max requests per second per IP (default: 50)
	HTTPRateBurst    int           // HTTP_RATE_BURST — burst size per IP (default: 50)
	HTTPReadTimeout  time.Duration // HTTP_READ_TIMEOUT — read timeout in seconds (default: 10)
	HTTPWriteTimeout time.Duration // HTTP_WRITE_TIMEOUT — write timeout in seconds (default: 30)
	HTTPIdleTimeout  time.Duration // HTTP_IDLE_TIMEOUT — idle timeout in seconds (default: 60)
	TrustedProxies   []net.IPNet   // HTTP_TRUSTED_PROXIES — comma-separated IPs/CIDRs; empty = trust all
}

// LoadBaseEnv reads the common MCP server environment variables.
func LoadBaseEnv() BaseEnv {
	return BaseEnv{
		APIKey:                   os.Getenv("X_API_KEY"),
		BaseURL:                  os.Getenv("BASE_URL"),
		OAuthAuthorizationServer: os.Getenv("OAUTH_AUTHORIZATION_SERVER"),
		OAuthAudience:            os.Getenv("OAUTH_AUDIENCE"),
		OAuthScopes:              os.Getenv("OAUTH_SCOPES"),
		OAuthClientSecret:        os.Getenv("OAUTH_CLIENT_SECRET"),
		HTTPHost:                 envString("HTTP_HOST", DefaultHTTPHost),
		HTTPPort:                 envInt("HTTP_PORT", DefaultHTTPPort),
		HTTPRateLimit:            envInt("HTTP_RATE_LIMIT", 50),
		HTTPRateBurst:            envInt("HTTP_RATE_BURST", 50),
		HTTPReadTimeout:          time.Duration(envInt("HTTP_READ_TIMEOUT", 10)) * time.Second,
		HTTPWriteTimeout:         time.Duration(envInt("HTTP_WRITE_TIMEOUT", 30)) * time.Second,
		HTTPIdleTimeout:          time.Duration(envInt("HTTP_IDLE_TIMEOUT", 60)) * time.Second,
		TrustedProxies:           parseTrustedProxies(os.Getenv("HTTP_TRUSTED_PROXIES")),
	}
}

// envInt reads an environment variable as an integer, returning def on missing or invalid values.
func envInt(key string, def int) int {
	s := os.Getenv(key)
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

// envString reads an environment variable, returning def when it is unset or empty.
func envString(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// httpAddr returns the address the HTTP transport listens on, from HTTP_HOST and HTTP_PORT.
func httpAddr() string {
	return net.JoinHostPort(
		envString("HTTP_HOST", DefaultHTTPHost),
		strconv.Itoa(envInt("HTTP_PORT", DefaultHTTPPort)),
	)
}

// OAuthScopesList returns the OAuth scopes as a string slice.
func (e BaseEnv) OAuthScopesList() []string {
	if e.OAuthScopes == "" {
		return nil
	}
	parts := strings.Split(e.OAuthScopes, ",")
	scopes := make([]string, 0, len(parts))
	for _, s := range parts {
		s = strings.TrimSpace(s)
		if s != "" {
			scopes = append(scopes, s)
		}
	}
	return scopes
}
