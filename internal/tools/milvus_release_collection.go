package tools

import (
	"context"

	"github.com/tailabs/mcp-milvus/internal/registry"
	"github.com/tailabs/mcp-milvus/internal/session"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

// Tool registrar
type ReleaseCollectionTool struct{}

func (t *ReleaseCollectionTool) GetTool() mcp.Tool {
	return NewMilvusReleaseCollectionTool()
}

func (t *ReleaseCollectionTool) GetHandler() server.ToolHandlerFunc {
	return MilvusReleaseCollectionHandler
}

func init() {
	registry.RegisterTool(&ReleaseCollectionTool{})
}

func NewMilvusReleaseCollectionTool() mcp.Tool {
	return mcp.NewTool("milvus_release_collection",
		mcp.WithDescription("Release a collection from memory."),
		mcp.WithString("collection_name",
			mcp.Required(),
			mcp.Description("Name of collection to release."),
		),
	)
}

func MilvusReleaseCollectionHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionClient := server.ClientSessionFromContext(ctx)
	cli, err := session.GetSessionManager().Get(sessionClient.SessionID())
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	collectionName, err := request.RequireString("collection_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	opt := milvusclient.NewReleaseCollectionOption(collectionName)
	if err := cli.ReleaseCollection(ctx, opt); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText("Collection released successfully."), nil
}
