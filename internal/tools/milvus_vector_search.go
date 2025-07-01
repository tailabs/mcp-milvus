package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/tailabs/mcp-milvus/internal/registry"
	"github.com/tailabs/mcp-milvus/internal/session"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

func NewMilvusVectorSearchTool() mcp.Tool {
	return mcp.NewTool("milvus_vector_search",
		mcp.WithDescription("Perform vector similarity search on a collection."),
		mcp.WithString("collection_name",
			mcp.Required(),
			mcp.Description("Name of the collection to search."),
		),
		mcp.WithString("vector",
			mcp.Required(),
			mcp.Description("Query vector as JSON array."),
		),
		mcp.WithString("vector_field",
			mcp.Description("Field containing vectors to search (default: 'vector')."),
		),
		mcp.WithString("limit",
			mcp.Description("Maximum number of results (default: 5)."),
		),
		mcp.WithString("output_fields",
			mcp.Description("Fields to include in results as JSON array."),
		),
		mcp.WithString("metric_type",
			mcp.Description("Distance metric (COSINE, L2, IP) (default: 'COSINE')."),
		),
		mcp.WithString("filter_expr",
			mcp.Description("Optional filter expression."),
		),
	)
}

func MilvusVectorSearchHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionClient := server.ClientSessionFromContext(ctx)
	cli, err := session.GetSessionManager().Get(sessionClient.SessionID())
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	collectionName, err := request.RequireString("collection_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	vectorStr, err := request.RequireString("vector")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var vector []float32
	if err := json.Unmarshal([]byte(vectorStr), &vector); err != nil {
		return mcp.NewToolResultError("Invalid vector JSON: " + err.Error()), nil
	}

	limitStr := request.GetString("limit", "5")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 5
	}

	var outputFields []string
	outputFieldsStr := request.GetString("output_fields", "")
	if outputFieldsStr != "" {
		if err := json.Unmarshal([]byte(outputFieldsStr), &outputFields); err != nil {
			return mcp.NewToolResultError("Invalid output_fields JSON: " + err.Error()), nil
		}
	}

	filterExpr := request.GetString("filter_expr", "")

	// Create vector data - Reference Python: data=[vector]
	vectorData := []entity.Vector{entity.FloatVector(vector)}

	// Use simplified search options
	opt := milvusclient.NewSearchOption(collectionName, limit, vectorData)

	if len(outputFields) > 0 {
		opt = opt.WithOutputFields(outputFields...)
	}

	if filterExpr != "" {
		opt = opt.WithFilter(filterExpr)
	}

	results, err := cli.Search(ctx, opt)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	output := fmt.Sprintf("Vector search results for collection '%s':\n\n", collectionName)

	// Simplified result processing
	if len(results) > 0 {
		resultSet := results[0]
		resultCount := len(resultSet.Scores)
		for i := 0; i < resultCount; i++ {
			result := map[string]interface{}{}

			// Get score
			if i < len(resultSet.Scores) {
				result["score"] = resultSet.Scores[i]
			}

			// Get ID
			if resultSet.IDs != nil {
				if id, idErr := resultSet.IDs.Get(i); idErr == nil {
					result["id"] = id
				}
			}

			// Get other fields
			if resultSet.Fields != nil {
				for _, column := range resultSet.Fields {
					if value, valueErr := column.Get(i); valueErr == nil {
						result[column.Name()] = value
					}
				}
			}

			output += fmt.Sprintf("%v\n\n", result)
		}
	} else {
		output += "No results found\n"
	}

	return mcp.NewToolResultText(output), nil
}

// Tool registrar
type VectorSearchTool struct{}

func (t *VectorSearchTool) GetTool() mcp.Tool {
	return NewMilvusVectorSearchTool()
}

func (t *VectorSearchTool) GetHandler() server.ToolHandlerFunc {
	return MilvusVectorSearchHandler
}

// Auto-register tool
func init() {
	registry.RegisterTool(&VectorSearchTool{})
}
