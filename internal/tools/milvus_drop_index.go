// milvus_drop_index.go
// Tool and handler for dropping index on Milvus collection fields.
package tools

import (
	"context"

	"github.com/tailabs/mcp-milvus/internal/registry"
	"github.com/tailabs/mcp-milvus/internal/session"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

// NewMilvusDropIndexTool creates a new tool for dropping an index from a collection
func NewMilvusDropIndexTool() mcp.Tool {
	return mcp.NewTool("milvus_drop_index",
		mcp.WithDescription("Drop an index from a collection."),
		mcp.WithString("collection_name",
			mcp.Required(),
			mcp.Description("Name of the collection."),
		),
		mcp.WithString("index_name",
			mcp.Required(),
			mcp.Description("Name of the index to drop."),
		),
	)
}

// MilvusDropIndexHandler handles the index drop request
func MilvusDropIndexHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionClient := server.ClientSessionFromContext(ctx)
	cli, err := session.GetSessionManager().Get(sessionClient.SessionID())
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	collectionName, err := request.RequireString("collection_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	indexName, err := request.RequireString("index_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	opt := milvusclient.NewDropIndexOption(collectionName, indexName)
	if err := cli.DropIndex(ctx, opt); err != nil {
		return mcp.NewToolResultError("Failed to drop index: " + err.Error()), nil
	}

	return mcp.NewToolResultText("Index '" + indexName + "' dropped successfully from collection '" + collectionName + "'"), nil
}

// Tool registrar
type DropIndexTool struct{}

func (t *DropIndexTool) GetTool() mcp.Tool {
	return NewMilvusDropIndexTool()
}

func (t *DropIndexTool) GetHandler() server.ToolHandlerFunc {
	return MilvusDropIndexHandler
}

func init() {
	registry.RegisterTool(&DropIndexTool{})
}
