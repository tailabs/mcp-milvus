package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tailabs/mcp-milvus/internal/middleware"
	"github.com/tailabs/mcp-milvus/internal/registry"
	"github.com/tailabs/mcp-milvus/internal/session"
	_ "github.com/tailabs/mcp-milvus/internal/tools"

	"github.com/mark3labs/mcp-go/server"
	"github.com/sirupsen/logrus"
)

func main() {
	// Initialize logging
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	// Setup session monitoring
	session.RegisterSessionEventCallbacks()

	// Create hooks
	hooks := session.NewSessionAwareHooks()

	// Create MCP server with enhanced features
	s := server.NewMCPServer(
		"mcp-milvus",
		"0.1.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(false, true),
		server.WithPromptCapabilities(true),
		server.WithRecovery(),
		server.WithHooks(hooks),
		server.WithToolHandlerMiddleware(middleware.Logging),
		server.WithToolHandlerMiddleware(middleware.Auth),
	)

	// Register all Milvus tools using global registry
	registry.RegisterAllTools(s)

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		logrus.Info("Starting MCP Milvus server...")

		// Start the SSE server
		sse := server.NewSSEServer(s)
		if err := sse.Start(":8080"); err != nil {
			logrus.Fatalf("Failed to start SSE server: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	logrus.Info("Received shutdown signal, gracefully shutting down...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Close session manager and cleanup all connections
	sessionManager := session.GetSessionManager()
	totalSessions := sessionManager.Size()
	logrus.WithField("total_sessions", totalSessions).Info("Closing session manager...")

	if err := sessionManager.Close(); err != nil {
		logrus.WithError(err).Error("Failed to close session manager")
	}

	// You could add server shutdown logic here if the server supports it
	// For now, we'll just log and exit
	select {
	case <-ctx.Done():
		logrus.Warn("Shutdown timeout")
	default:
		logrus.Info("Server shutdown successfully")
	}
}
