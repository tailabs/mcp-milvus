package tools

import (
	"context"
	"fmt"

	"github.com/tailabs/mcp-milvus/internal/registry"
	"github.com/tailabs/mcp-milvus/internal/session"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewMilvusConnectorTool returns a tool for connecting to Milvus with detailed parameters.
func NewMilvusConnectorTool() mcp.Tool {
	return mcp.NewTool("milvus_connector",
		mcp.WithDescription("Connect to a Milvus server instance with authentication and database selection."),
		mcp.WithString("address",
			mcp.Required(),
			mcp.Description("The URI address of the Milvus server, e.g., 'http://localhost:19530'."),
		),
		mcp.WithString("token",
			mcp.Description("Authentication credentials in the format 'username:password'."),
		),
		mcp.WithString("db_name",
			mcp.DefaultString("default"),
			mcp.Description("The name of the database to connect to, e.g., 'default'."),
		),
	)
}

// MilvusConnectorHandler handles the milvus_connector tool call.
func MilvusConnectorHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var connConfig session.ConnConfig
	if err := request.BindArguments(&connConfig); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	sessionClient := server.ClientSessionFromContext(ctx)
	if err := session.GetSessionManager().Set(sessionClient.SessionID(), &connConfig); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Connected to Milvus successfully, database: %s", connConfig.DBName)), nil
}

type ConnectorTool struct{}

func (t *ConnectorTool) GetTool() mcp.Tool {
	return NewMilvusConnectorTool()
}

func (t *ConnectorTool) GetHandler() server.ToolHandlerFunc {
	return MilvusConnectorHandler
}

// Auto-register tool
func init() {
	registry.RegisterTool(&ConnectorTool{})
}
