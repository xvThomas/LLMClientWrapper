package mcpserver

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/pixime-net/talk-libs/logger"
)

// Option configures an App. Use the With* functions to create options.
type Option func(*App)

// WithAPIKey enables X-API-Key header authentication.
func WithAPIKey(key string) Option {
	return func(a *App) { a.apiKey = &key }
}

// WithOAuth enables OAuth 2.0 Bearer token authentication.
func WithOAuth(cfg *OAuthConfig) Option {
	return func(a *App) { a.oauth = cfg }
}

// WithTools registers tools on the MCP server.
func WithTools(tools ...ToolRegistrar) Option {
	return func(a *App) { a.tools = append(a.tools, tools...) }
}

// App is a reusable MCP server runner that handles CLI flags, transport
// routing (stdio / HTTP), and server creation.
//
// Create with NewApp and configure with functional options:
//
//	app := mcpserver.NewApp("my-mcp", "1.0.0",
//	    mcpserver.WithAPIKey(env.APIKey),
//	    mcpserver.WithTools(mcpserver.RegisterTool(myTool)),
//	)
//	app.Run()
type App struct {
	name     string
	version  string
	tools    []ToolRegistrar
	prompts  []PromptRegistrar
	apiKey   *string             // optional: X-API-Key header authentication
	oauth    *OAuthConfig        // optional: OAuth Bearer token authentication
	security *HTTPSecurityConfig // optional: HTTP security settings
}

// NewApp creates an App configured with the given options.
func NewApp(name, version string, opts ...Option) *App {
	a := &App{name: name, version: version}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// Run parses CLI flags and starts the server using the selected transport.
func (a *App) Run() {
	log := logger.GetLogger()

	transport := flag.String("transport", "stdio", "transport to use: stdio | http")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\nOptions:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nThe HTTP transport listens on $HTTP_HOST:$HTTP_PORT (default %s:%d).\n",
			DefaultHTTPHost, DefaultHTTPPort)
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s --transport stdio\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --transport http\n", os.Args[0])
	}
	flag.Parse()

	log.Info("MCP Server", "name", a.name, "version", a.version)

	toolNames := make([]string, len(a.tools))
	for i, t := range a.tools {
		toolNames[i] = t.Name
	}
	log.Info("Available tools", "count", len(toolNames), "tools", toolNames)

	promptNames := make([]string, len(a.prompts))
	for i, p := range a.prompts {
		promptNames[i] = p.Name
	}
	if len(promptNames) > 0 {
		log.Info("Available prompts", "count", len(promptNames), "prompts", promptNames)
	}

	switch *transport {
	case "stdio":
		a.runStdio()
	case "http":
		a.runHTTP(httpAddr())
	default:
		log.Error("unknown transport", "transport", *transport)
		flag.Usage()
		os.Exit(1)
	}
}

func (a *App) newServer() *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: a.name, Version: a.version}, nil)
	for _, t := range a.tools {
		t.Register(s)
	}
	for _, p := range a.prompts {
		p.Register(s)
	}
	return s
}

func (a *App) runStdio() {
	log := logger.GetLogger()
	s := a.newServer()
	log.Info("Stdio server running")
	if err := s.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Error("stdio server failed", "error", err)
		os.Exit(1)
	}
}

// WithBaseEnvHTTPSecurity returns an Option that configures HTTP security
// settings from the common BaseEnv fields.
func WithBaseEnvHTTPSecurity(env BaseEnv) Option {
	return WithHTTPSecurity(HTTPSecurityConfig{
		RateLimit:      env.HTTPRateLimit,
		RateBurst:      env.HTTPRateBurst,
		ReadTimeout:    env.HTTPReadTimeout,
		WriteTimeout:   env.HTTPWriteTimeout,
		IdleTimeout:    env.HTTPIdleTimeout,
		TrustedProxies: env.TrustedProxies,
	})
}

// WithBaseEnvAuth returns the authentication Options derived from the common
// BaseEnv fields (API key and/or OAuth). The returned slice is empty when
// neither is configured.
func WithBaseEnvAuth(env BaseEnv) []Option {
	var opts []Option
	if env.APIKey != "" {
		opts = append(opts, WithAPIKey(env.APIKey))
	}
	if env.OAuthAuthorizationServer != "" {
		cfg := &OAuthConfig{
			AuthorizationServerURL: env.OAuthAuthorizationServer,
			ResourceBaseURL:        env.BaseURL,
			Scopes:                 env.OAuthScopesList(),
			TokenVerifier: NewJWKSTokenVerifier(JWKSVerifierConfig{
				IssuerURL: env.OAuthAuthorizationServer,
				Audience:  env.OAuthAudience,
			}),
		}
		if env.OAuthAudience != "" {
			cfg.ASProxy = &ASProxyConfig{
				Audience:     env.OAuthAudience,
				ClientSecret: env.OAuthClientSecret,
			}
		}
		opts = append(opts, WithOAuth(cfg))
	}
	return opts
}
