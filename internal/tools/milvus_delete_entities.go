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

func NewMilvusDeleteEntitiesTool() mcp.Tool {
	return mcp.NewTool("milvus_delete_entities",
		mcp.WithDescription("Delete entities from a collection based on filter expression."),
		mcp.WithString("collection_name",
			mcp.Required(),
			mcp.Description("Name of collection."),
		),
		mcp.WithString("filter_expr",
			mcp.Required(),
			mcp.Description("Filter expression to select entities to delete."),
		),
	)
}

func MilvusDeleteEntitiesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionClient := server.ClientSessionFromContext(ctx)
	cli, err := session.GetSessionManager().Get(sessionClient.SessionID())
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	collectionName, err := request.RequireString("collection_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	filterExpr, err := request.RequireString("filter_expr")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	opt := milvusclient.NewDeleteOption(collectionName).WithExpr(filterExpr)
	result, err := cli.Delete(ctx, opt)

	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Delete result: %v", result)), nil
}

// Tool registrar
type DeleteEntitiesTool struct{}

func (t *DeleteEntitiesTool) GetTool() mcp.Tool {
	return NewMilvusDeleteEntitiesTool()
}

func (t *DeleteEntitiesTool) GetHandler() server.ToolHandlerFunc {
	return MilvusDeleteEntitiesHandler
}

func init() {
	registry.RegisterTool(&DeleteEntitiesTool{})
}
