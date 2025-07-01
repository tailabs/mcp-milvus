package tools

import (
	"context"
	"fmt"

	"github.com/tailabs/mcp-milvus/internal/registry"
	"github.com/tailabs/mcp-milvus/internal/session"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

// NewMilvusCreateDatabaseTool creates a new tool for creating databases
func NewMilvusCreateDatabaseTool() mcp.Tool {
	return mcp.NewTool("milvus_create_database",
		mcp.WithDescription("Create a new database in Milvus."),
		mcp.WithString("database_name",
			mcp.Required(),
			mcp.Description("Name of the database to create."),
		),
	)
}

// MilvusCreateDatabaseHandler handles the database creation request
func MilvusCreateDatabaseHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionClient := server.ClientSessionFromContext(ctx)
	cli, err := session.GetSessionManager().Get(sessionClient.SessionID())
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	databaseName, err := request.RequireString("database_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Create database
	opt := milvusclient.NewCreateDatabaseOption(databaseName)
	if err := cli.CreateDatabase(ctx, opt); err != nil {
		return mcp.NewToolResultError("Failed to create database: " + err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Database '%s' created successfully", databaseName)), nil
}

// Tool registrar
type CreateDatabaseTool struct{}

func (t *CreateDatabaseTool) GetTool() mcp.Tool {
	return NewMilvusCreateDatabaseTool()
}

func (t *CreateDatabaseTool) GetHandler() server.ToolHandlerFunc {
	return MilvusCreateDatabaseHandler
}

func init() {
	registry.RegisterTool(&CreateDatabaseTool{})
}
