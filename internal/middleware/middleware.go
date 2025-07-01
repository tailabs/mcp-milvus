package middleware

import (
	"context"
	"time"

	"github.com/tailabs/mcp-milvus/internal/session"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sirupsen/logrus"
)

func Logging(next server.ToolHandlerFunc) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (cr *mcp.CallToolResult, err error) {
		start := time.Now()
		sessionID := server.ClientSessionFromContext(ctx)

		l := logrus.WithFields(logrus.Fields{
			"session": sessionID.SessionID(),
			"tool":    req.Params.Name,
		})

		defer func() {
			duration := time.Since(start)
			if err != nil {
				l.WithField("duration", duration).Errorf("Tool call failed, %v", err)
			} else if cr != nil && cr.IsError {
				l.WithField("duration", duration).Errorf("Tool call failed, %#+v", cr)
			} else {
				l.WithField("duration", duration).Info("Tool call completed")
			}
		}()

		return next(ctx, req)
	}
}

func Auth(next server.ToolHandlerFunc) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sessionClient := server.ClientSessionFromContext(ctx)
		if sessionClient == nil || sessionClient.SessionID() == "" {
			return mcp.NewToolResultError("must provide an available session id"), nil
		}

		if req.Params.Name == "milvus_connector" {
			return next(ctx, req)
		}

		_, err := session.GetSessionManager().Get(sessionClient.SessionID())
		if err != nil {
			return mcp.NewToolResultError("auth first, please call milvus_connector tool"), nil
		}
		return next(ctx, req)
	}
}
