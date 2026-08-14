package main

import (
	"os"

	"github.com/pixime-net/talkbackend/mcp-ign-nav/internal/config"
	"github.com/pixime-net/talkbackend/mcp-ign-nav/internal/tools"
	"github.com/pixime-net/talkbackend/talk-libs/logger"
	"github.com/pixime-net/talkbackend/talk-libs/mcpserver"
	"github.com/pixime-net/talkbackend/talk-libs/version"
	"golang.org/x/time/rate"
)

func main() {
	log := logger.GetLogger()

	env, err := config.LoadServerEnv(".env")
	if err != nil {
		log.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	app := buildApp(env)
	app.Run()
}

// buildApp creates the MCP server application from the given configuration.
func buildApp(env *config.ServerEnv) *mcpserver.App {
	// Shared rate limiter for IGN Géoplateforme endpoints (50 req/s).
	ignLimiter := rate.NewLimiter(rate.Limit(50), 50)

	// Rate limiter for navigation endpoint (5 req/s).
	navLimiter := rate.NewLimiter(rate.Limit(5), 5)

	opts := []mcpserver.Option{
		mcpserver.WithTools(
			mcpserver.RegisterTool(tools.NewReverseGeocodingTool(ignLimiter)),
			mcpserver.RegisterTool(tools.NewGeocodingTool(ignLimiter)),
			mcpserver.RegisterTool(tools.NewRouteTool(navLimiter, env.GetGeoJSONGeometry)),
			mcpserver.RegisterTool(tools.NewDistanceTimeTool(navLimiter)),
		),
		mcpserver.WithBaseEnvHTTPSecurity(env.BaseEnv),
	}
	opts = append(opts, mcpserver.WithBaseEnvAuth(env.BaseEnv)...)
	return mcpserver.NewApp("ign-nav-mcp", version.Version, opts...)
}
