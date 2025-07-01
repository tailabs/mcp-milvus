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

// NewMilvusDropCollectionTool creates a new tool for dropping collections
func NewMilvusDropCollectionTool() mcp.Tool {
	return mcp.NewTool("milvus_drop_collection",
		mcp.WithDescription("Drop a collection and all its data from Milvus."),
		mcp.WithString("collection_name",
			mcp.Required(),
			mcp.Description("Name of the collection to drop."),
		),
	)
}

// MilvusDropCollectionHandler handles the collection dropping request
func MilvusDropCollectionHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionClient := server.ClientSessionFromContext(ctx)
	cli, err := session.GetSessionManager().Get(sessionClient.SessionID())
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	collectionName, err := request.RequireString("collection_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Drop collection
	opt := milvusclient.NewDropCollectionOption(collectionName)
	if err := cli.DropCollection(ctx, opt); err != nil {
		return mcp.NewToolResultError("Failed to drop collection: " + err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Collection '%s' dropped successfully", collectionName)), nil
}

// Tool registrar
type DropCollectionTool struct{}

func (t *DropCollectionTool) GetTool() mcp.Tool {
	return NewMilvusDropCollectionTool()
}

func (t *DropCollectionTool) GetHandler() server.ToolHandlerFunc {
	return MilvusDropCollectionHandler
}

func init() {
	registry.RegisterTool(&DropCollectionTool{})
}
