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

// Tool registrar
type UseDatabaseTool struct{}

func (t *UseDatabaseTool) GetTool() mcp.Tool {
	return NewMilvusUseDatabaseTool()
}

func (t *UseDatabaseTool) GetHandler() server.ToolHandlerFunc {
	return MilvusUseDatabaseHandler
}

func init() {
	registry.RegisterTool(&UseDatabaseTool{})
}

func NewMilvusUseDatabaseTool() mcp.Tool {
	return mcp.NewTool("milvus_use_database",
		mcp.WithDescription("Switch to a specific database."),
		mcp.WithString("database_name",
			mcp.Required(),
			mcp.Description("Name of the database to switch to."),
		),
	)
}

func MilvusUseDatabaseHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionClient := server.ClientSessionFromContext(ctx)
	cli, err := session.GetSessionManager().Get(sessionClient.SessionID())
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	databaseName, err := request.RequireString("database_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	opt := milvusclient.NewUseDatabaseOption(databaseName)
	err = cli.UseDatabase(ctx, opt)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully switched to database: %s", databaseName)), nil
}
