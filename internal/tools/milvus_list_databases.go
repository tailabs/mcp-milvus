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

// NewMilvusListDatabasesTool returns a tool for listing all databases in Milvus.
func NewMilvusListDatabasesTool() mcp.Tool {
	return mcp.NewTool("milvus_list_databases",
		mcp.WithDescription("List all databases in the connected Milvus instance."),
	)
}

// MilvusListDatabasesHandler handles the milvus_list_databases tool call.
func MilvusListDatabasesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionClient := server.ClientSessionFromContext(ctx)
	cli, err := session.GetSessionManager().Get(sessionClient.SessionID())
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// The actual method to list databases may need to be updated to match your milvus client
	dbs, err := cli.ListDatabase(ctx, milvusclient.NewListDatabaseOption())
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Databases: %v", dbs)), nil
}

// Tool registrar
type ListDatabasesTool struct{}

func (t *ListDatabasesTool) GetTool() mcp.Tool {
	return NewMilvusListDatabasesTool()
}

func (t *ListDatabasesTool) GetHandler() server.ToolHandlerFunc {
	return MilvusListDatabasesHandler
}

func init() {
	registry.RegisterTool(&ListDatabasesTool{})
}
