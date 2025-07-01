package tools

import (
	"context"
	"fmt"
	"strconv"

	"github.com/tailabs/mcp-milvus/internal/registry"
	"github.com/tailabs/mcp-milvus/internal/session"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

func NewMilvusLoadCollectionTool() mcp.Tool {
	return mcp.NewTool("milvus_load_collection",
		mcp.WithDescription("Load a collection into memory for search and query."),
		mcp.WithString("collection_name",
			mcp.Required(),
			mcp.Description("Name of collection to load."),
		),
		mcp.WithString("replica_number",
			mcp.Description("Number of replicas (default: 1)."),
		),
	)
}

func MilvusLoadCollectionHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionClient := server.ClientSessionFromContext(ctx)
	cli, err := session.GetSessionManager().Get(sessionClient.SessionID())
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	collectionName, err := request.RequireString("collection_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	replicaNumber := 1
	replicaNumberStr := request.GetString("replica_number", "1")
	if replicaNumberStr != "" {
		if parsed, parseErr := strconv.Atoi(replicaNumberStr); parseErr == nil {
			replicaNumber = parsed
		}
	}

	opt := milvusclient.NewLoadCollectionOption(collectionName)
	task, err := cli.LoadCollection(ctx, opt)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if err := task.Await(ctx); err != nil {
		return mcp.NewToolResultError("Load collection failed: " + err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Collection '%s' loaded successfully with %d replica(s)", collectionName, replicaNumber)), nil
}

// Tool registrar
type LoadCollectionTool struct{}

func (t *LoadCollectionTool) GetTool() mcp.Tool {
	return NewMilvusLoadCollectionTool()
}

func (t *LoadCollectionTool) GetHandler() server.ToolHandlerFunc {
	return MilvusLoadCollectionHandler
}

func init() {
	registry.RegisterTool(&LoadCollectionTool{})
}
