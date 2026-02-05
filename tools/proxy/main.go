package main

import (
	"context"
	"flag"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	listenAddr := flag.String("listen", ":19132", "Address for the Minecraft proxy listener")
	invite := flag.String("invite", "", "Realm invite code (overrides REALM_INVITE env / .realm-invite file)")
	flag.Parse()

	// Log to file (stdout is MCP stdio, stderr may not be visible)
	logFile, err := os.OpenFile("proxy.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		os.Exit(1)
	}
	defer logFile.Close()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.MultiWriter(os.Stderr, logFile), &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	// Load Xbox Live token
	tokenSource, err := getTokenSource()
	if err != nil {
		slog.Error("authentication failed", "error", err)
		os.Exit(1)
	}

	// Resolve realm invite code
	inviteCode := *invite
	if inviteCode == "" {
		var err error
		inviteCode, err = getRealmInvite()
		if err != nil {
			slog.Error("realm invite error", "error", err)
			os.Exit(1)
		}
	}

	// Create game state
	state := NewGameState()

	// Create MCP server
	mcpServer := server.NewMCPServer(
		"minecraft-proxy",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	// Register all tools
	registerQueryTools(mcpServer, state)
	registerActionTools(mcpServer, state)

	// Setup context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		slog.Info("shutting down...")
		cancel()
	}()

	// Start proxy in background goroutine
	go startProxy(ctx, *listenAddr, inviteCode, tokenSource, state)

	// Serve MCP over stdio (blocks)
	slog.Info("MCP server starting on stdio")
	if err := server.ServeStdio(mcpServer); err != nil {
		slog.Error("MCP server error", "error", err)
		os.Exit(1)
	}
}
