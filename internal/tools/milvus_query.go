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
	"github.com/milvus-io/milvus/client/v2/column"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"github.com/samber/lo"
)

func NewMilvusQueryTool() mcp.Tool {
	return mcp.NewTool("milvus_query",
		mcp.WithDescription("Query collection using filter expressions."),
		mcp.WithString("collection_name",
			mcp.Required(),
			mcp.Description("Name of the collection to query."),
		),
		mcp.WithString("filter_expr",
			mcp.Required(),
			mcp.Description("Filter expression (e.g. 'age > 20')."),
		),
		mcp.WithString("output_fields",
			mcp.Description("Fields to include in results as JSON array."),
		),
		mcp.WithString("limit",
			mcp.Description("Maximum number of results (default: 10)."),
		),
	)
}

func MilvusQueryHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	var outputFields []string
	outputFieldsStr := request.GetString("output_fields", "")
	if outputFieldsStr != "" {
		if err := json.Unmarshal([]byte(outputFieldsStr), &outputFields); err != nil {
			return mcp.NewToolResultError("Invalid output_fields JSON: " + err.Error()), nil
		}
	}

	limit := 10
	limitStr := request.GetString("limit", "10")
	if limitStr != "" {
		if parsedLimit, parseErr := strconv.Atoi(limitStr); parseErr == nil {
			limit = parsedLimit
		}
	}

	opt := milvusclient.NewQueryOption(collectionName).
		WithFilter(filterExpr).
		WithOutputFields(outputFields...).
		WithLimit(limit)
	results, err := cli.Query(ctx, opt)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	queryResultMaps, err := resultSetToMaps(results)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	outputResult, err := json.MarshalIndent(queryResultMaps, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("Failed to format query results: " + err.Error()), nil
	}

	output := fmt.Sprintf("Query results for '%s' in collection '%s':\n\n", filterExpr, collectionName)
	output += fmt.Sprintf("Results: %s\n", string(outputResult))

	return mcp.NewToolResultText(output), nil
}

func resultSetToMaps(resultSet milvusclient.ResultSet) ([]map[string]any, error) {
	if resultSet.ResultCount == 0 {
		return []map[string]any{}, nil
	}

	// Collect field names and columns
	fieldNames := lo.Map(resultSet.Fields, func(f column.Column, _ int) string {
		return f.Name()
	})
	fieldColumns := lo.Map(fieldNames, func(name string, _ int) column.Column {
		return resultSet.GetColumn(name)
	})

	data := make([]map[string]any, 0, resultSet.ResultCount)
	for i := 0; i < resultSet.ResultCount; i++ {
		row := make(map[string]any, len(fieldNames))
		for j, col := range fieldColumns {
			val, err := col.Get(i)
			if err != nil {
				return nil, fmt.Errorf("error at row %d col %s: %v", i, fieldNames[j], err)
			}
			row[fieldNames[j]] = val
		}
		data = append(data, row)
	}
	return data, nil
}

// Tool registrar
type QueryTool struct{}

func (t *QueryTool) GetTool() mcp.Tool {
	return NewMilvusQueryTool()
}

func (t *QueryTool) GetHandler() server.ToolHandlerFunc {
	return MilvusQueryHandler
}

// Auto-register tool
func init() {
	registry.RegisterTool(&QueryTool{})
}
