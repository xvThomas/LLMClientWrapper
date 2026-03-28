package version

// Version is injected at build time via:
//
// go build -ldflags "-X llmclientwrapper/src/internal/version.Version=v1.2.3"
var Version = "v0.0.0"
