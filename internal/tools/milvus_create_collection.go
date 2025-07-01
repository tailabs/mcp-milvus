package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tailabs/mcp-milvus/internal/registry"
	"github.com/tailabs/mcp-milvus/internal/schema"
	"github.com/tailabs/mcp-milvus/internal/session"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/milvus-io/milvus/client/v2/index"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

// NewMilvusCreateCollectionTool creates a new tool for creating Milvus collections
func NewMilvusCreateCollectionTool() mcp.Tool {
	return mcp.NewTool("milvus_create_collection",
		mcp.WithDescription("Create a new collection with specified schema."),
		mcp.WithString("collection_name",
			mcp.Required(),
			mcp.Description("Name for the new collection."),
		),
		mcp.WithString("collection_schema",
			mcp.Required(),
			mcp.Description("Collection schema definition as JSON. Example: {\"auto_id\": false, \"enable_dynamic_field\": true, \"fields\": [{\"name\": \"id\", \"data_type\": \"Int64\", \"is_primary_key\": true}, {\"name\": \"vector\", \"data_type\": \"FloatVector\", \"dim\": 128}]}"),
		),
		mcp.WithString("index_params",
			mcp.Description("Optional index parameters as JSON array. Example: [{\"field_name\": \"vector\", \"index_type\": \"AUTOINDEX\", \"metric_type\": \"COSINE\", \"params\": {}}]"),
		),
	)
}

// MilvusCreateCollectionHandler handles the collection creation request
func MilvusCreateCollectionHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionClient := server.ClientSessionFromContext(ctx)
	cli, err := session.GetSessionManager().Get(sessionClient.SessionID())
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	collectionName, err := request.RequireString("collection_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	schemaStr, err := request.RequireString("collection_schema")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Parse schema to map[string]any first
	var schemaMap map[string]any
	if err := json.Unmarshal([]byte(schemaStr), &schemaMap); err != nil {
		return mcp.NewToolResultError("Invalid collection_schema JSON: " + err.Error()), nil
	}

	// Build schema from map
	collectionSchema, err := schema.BuildSchemaFromMap(schemaMap)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to build schema: %v", err)), nil
	}

	// Create collection option
	opt := milvusclient.NewCreateCollectionOption(collectionName, collectionSchema)

	// Create collection
	if err := cli.CreateCollection(ctx, opt); err != nil {
		return mcp.NewToolResultError("Failed to create collection: " + err.Error()), nil
	}

	// Handle optional index parameters
	indexParamsStr := request.GetString("index_params", "")
	if indexParamsStr != "" {
		var indexConfigs []map[string]any
		if err := json.Unmarshal([]byte(indexParamsStr), &indexConfigs); err != nil {
			return mcp.NewToolResultError("Invalid index_params JSON: " + err.Error()), nil
		}

		// Create index for each config
		for _, cfg := range indexConfigs {
			field, _ := cfg["field_name"].(string)
			indexType, _ := cfg["index_type"].(string)
			metricType, _ := cfg["metric_type"].(string)
			params, _ := cfg["params"].(map[string]any)

			// Build index params
			indexParams := map[string]string{}
			for k, v := range params {
				indexParams[k] = fmt.Sprintf("%v", v)
			}
			// Add required index_type and metric_type
			indexParams["index_type"] = indexType
			indexParams["metric_type"] = metricType

			// Create generic index
			idx := index.NewGenericIndex("", indexParams)
			opt := milvusclient.NewCreateIndexOption(collectionName, field, idx)
			task, err := cli.CreateIndex(ctx, opt)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("CreateIndex failed for field %s: %v", field, err)), nil
			}
			// Wait for index creation to finish
			if err := task.Await(ctx); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("CreateIndex await failed for field %s: %v", field, err)), nil
			}
		}
	}

	// Count fields from schema map
	fieldsData, _ := schemaMap["fields"].([]any)
	return mcp.NewToolResultText(fmt.Sprintf("Collection '%s' created successfully with %d fields",
		collectionName, len(fieldsData))), nil
}

// Tool registrar
type CreateCollectionTool struct{}

func (t *CreateCollectionTool) GetTool() mcp.Tool {
	return NewMilvusCreateCollectionTool()
}

func (t *CreateCollectionTool) GetHandler() server.ToolHandlerFunc {
	return MilvusCreateCollectionHandler
}

func init() {
	registry.RegisterTool(&CreateCollectionTool{})
}
