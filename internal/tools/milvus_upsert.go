package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tailabs/mcp-milvus/internal/registry"
	"github.com/tailabs/mcp-milvus/internal/session"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

// NewMilvusUpsertTool creates a new tool for upserting data
func NewMilvusUpsertTool() mcp.Tool {
	return mcp.NewTool("milvus_upsert",
		mcp.WithDescription("Upsert (insert or update) data into a collection."),
		mcp.WithString("collection_name",
			mcp.Required(),
			mcp.Description("Name of the collection to upsert data into."),
		),
		mcp.WithString("data",
			mcp.Required(),
			mcp.Description("List of dictionaries, each representing a record to upsert."),
		),
		mcp.WithString("partition_name",
			mcp.Description("Name of the partition to upsert data into (optional, defaults to default partition)."),
		),
	)
}

// MilvusUpsertHandler handles the upsert request
func MilvusUpsertHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionClient := server.ClientSessionFromContext(ctx)
	cli, err := session.GetSessionManager().Get(sessionClient.SessionID())
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	collectionName, err := request.RequireString("collection_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	dataStr, err := request.RequireString("data")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	partitionName := request.GetString("partition_name", "")

	// Parse the data from JSON string
	var data []interface{}
	if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
		return mcp.NewToolResultError("Invalid data JSON: " + err.Error()), nil
	}

	if len(data) == 0 {
		return mcp.NewToolResultError("Data cannot be empty"), nil
	}

	// Transform data using the same logic as insert
	transformedData, err := transformDataForCollection(ctx, cli, collectionName, data)
	if err != nil {
		return mcp.NewToolResultError("Failed to transform data: " + err.Error()), nil
	}

	// Perform upsert using row-based approach similar to insert
	opt := milvusclient.NewRowBasedInsertOption(collectionName, transformedData...)
	if partitionName != "" {
		opt.WithPartition(partitionName)
	}

	result, err := cli.Upsert(ctx, opt)
	if err != nil {
		return mcp.NewToolResultError("Failed to upsert data: " + err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Upserted %d records successfully. Upsert count: %d", len(transformedData), result.UpsertCount)), nil
}

// Tool registrar
type UpsertTool struct{}

func (t *UpsertTool) GetTool() mcp.Tool {
	return NewMilvusUpsertTool()
}

func (t *UpsertTool) GetHandler() server.ToolHandlerFunc {
	return MilvusUpsertHandler
}

func init() {
	registry.RegisterTool(&UpsertTool{})
}
