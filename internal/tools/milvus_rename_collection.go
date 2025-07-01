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

// NewMilvusRenameCollectionTool creates a new tool for renaming collections
func NewMilvusRenameCollectionTool() mcp.Tool {
	return mcp.NewTool("milvus_rename_collection",
		mcp.WithDescription("Rename an existing collection."),
		mcp.WithString("old_collection_name",
			mcp.Required(),
			mcp.Description("Current name of the collection to rename."),
		),
		mcp.WithString("new_collection_name",
			mcp.Required(),
			mcp.Description("New name for the collection."),
		),
	)
}

// MilvusRenameCollectionHandler handles the collection renaming request
func MilvusRenameCollectionHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionClient := server.ClientSessionFromContext(ctx)
	cli, err := session.GetSessionManager().Get(sessionClient.SessionID())
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	oldCollectionName, err := request.RequireString("old_collection_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	newCollectionName, err := request.RequireString("new_collection_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Rename collection
	opt := milvusclient.NewRenameCollectionOption(oldCollectionName, newCollectionName)
	if err := cli.RenameCollection(ctx, opt); err != nil {
		return mcp.NewToolResultError("Failed to rename collection: " + err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Collection '%s' renamed to '%s' successfully", oldCollectionName, newCollectionName)), nil
}

// Tool registrar
type RenameCollectionTool struct{}

func (t *RenameCollectionTool) GetTool() mcp.Tool {
	return NewMilvusRenameCollectionTool()
}

func (t *RenameCollectionTool) GetHandler() server.ToolHandlerFunc {
	return MilvusRenameCollectionHandler
}

func init() {
	registry.RegisterTool(&RenameCollectionTool{})
}
