// milvus_create_index.go
// Tool and handler for creating index on Milvus collection fields.
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tailabs/mcp-milvus/internal/registry"
	"github.com/tailabs/mcp-milvus/internal/session"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/milvus-io/milvus/client/v2/index"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

// NewMilvusCreateIndexTool creates a new tool for creating an index on an existing collection
func NewMilvusCreateIndexTool() mcp.Tool {
	return mcp.NewTool("milvus_create_index",
		mcp.WithDescription("Create an index for a collection field."),
		mcp.WithString("collection_name",
			mcp.Required(),
			mcp.Description("Name of the collection."),
		),
		mcp.WithString("field_name",
			mcp.Required(),
			mcp.Description("Name of the field to create index for."),
		),
		mcp.WithString("index_type",
			mcp.Required(),
			mcp.Description("Type of the index, e.g. IVF_FLAT, HNSW, etc."),
		),
		mcp.WithString("metric_type",
			mcp.Required(),
			mcp.Description("Metric type, e.g. COSINE, L2, etc."),
		),
		mcp.WithString("params",
			mcp.Description("Index parameters as JSON, e.g. {\"nlist\": 128}"),
		),
	)
}

// MilvusCreateIndexHandler handles the index creation request
func MilvusCreateIndexHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionClient := server.ClientSessionFromContext(ctx)
	cli, err := session.GetSessionManager().Get(sessionClient.SessionID())
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	collectionName, err := request.RequireString("collection_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	fieldName, err := request.RequireString("field_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	indexType, err := request.RequireString("index_type")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	metricType, err := request.RequireString("metric_type")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	paramsStr := request.GetString("params", "")
	params := map[string]any{}
	if paramsStr != "" {
		if err := json.Unmarshal([]byte(paramsStr), &params); err != nil {
			return mcp.NewToolResultError("Invalid params JSON: " + err.Error()), nil
		}
	}

	// Build index params (case-insensitive keys)
	indexParams := map[string]string{}
	for k, v := range params {
		indexParams[k] = fmt.Sprintf("%v", v)
	}
	indexParams["index_type"] = indexType
	indexParams["metric_type"] = metricType

	// Create generic index
	idx := index.NewGenericIndex("", indexParams)
	opt := milvusclient.NewCreateIndexOption(collectionName, fieldName, idx)
	task, err := cli.CreateIndex(ctx, opt)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("CreateIndex failed: %v", err)), nil
	}
	if err := task.Await(ctx); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("CreateIndex await failed: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Index created successfully for collection '%s', field '%s'", collectionName, fieldName)), nil
}

// Tool registrar
type CreateIndexTool struct{}

func (t *CreateIndexTool) GetTool() mcp.Tool {
	return NewMilvusCreateIndexTool()
}

func (t *CreateIndexTool) GetHandler() server.ToolHandlerFunc {
	return MilvusCreateIndexHandler
}

func init() {
	registry.RegisterTool(&CreateIndexTool{})
}
