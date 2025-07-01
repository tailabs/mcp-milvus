package registry

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type ToolRegistrar interface {
	GetTool() mcp.Tool
	GetHandler() server.ToolHandlerFunc
}

var globalToolRegistry = make([]ToolRegistrar, 0)

func RegisterTool(tool ToolRegistrar) {
	globalToolRegistry = append(globalToolRegistry, tool)
}

func RegisterAllTools(s *server.MCPServer) {
	for _, tool := range globalToolRegistry {
		s.AddTool(tool.GetTool(), tool.GetHandler())
	}
}
