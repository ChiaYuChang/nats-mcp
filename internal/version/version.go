// Package version provides the application version shared by the CLI and MCP.
package version

// Version is overridden at build time with -ldflags -X for release builds.
var Version = "dev"
