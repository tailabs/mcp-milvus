package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/tailabs/mcp-milvus/internal/registry"
	"github.com/tailabs/mcp-milvus/internal/session"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

func NewMilvusListCollectionsTool() mcp.Tool {
	return mcp.NewTool("milvus_list_collections",
		mcp.WithDescription("List all collections in the database."),
	)
}

func MilvusListCollectionsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionClient := server.ClientSessionFromContext(ctx)
	cli, err := session.GetSessionManager().Get(sessionClient.SessionID())
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	collections, err := cli.ListCollections(ctx, milvusclient.NewListCollectionOption())
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Collections in database:\n%s", strings.Join(collections, ", "))), nil
}

// Tool registrar
type ListCollectionsTool struct{}

func (t *ListCollectionsTool) GetTool() mcp.Tool {
	return NewMilvusListCollectionsTool()
}

func (t *ListCollectionsTool) GetHandler() server.ToolHandlerFunc {
	return MilvusListCollectionsHandler
}

// Auto-register tool
func init() {
	registry.RegisterTool(&ListCollectionsTool{})
}
