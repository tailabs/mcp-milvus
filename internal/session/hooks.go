package session

import (
	"context"
	"time"

	"github.com/mark3labs/mcp-go/server"
	"github.com/sirupsen/logrus"
)

// RegisterSessionEventCallbacks registers session event callbacks
func RegisterSessionEventCallbacks() {
	sessionManager := GetSessionManager()

	// Add session monitoring callback
	sessionManager.AddEventCallback(func(event SessionEvent, sessionID string, state *SessionState) {
		switch event {
		case SessionCreated:
			logrus.WithFields(logrus.Fields{
				"event":      event,
				"session_id": sessionID,
				"address":    state.ConnConfig.Address,
				"database":   state.ConnConfig.DBName,
			}).Info("Session created")
		case SessionRemoved:
			logrus.WithFields(logrus.Fields{
				"event":        event,
				"session_id":   sessionID,
				"access_count": state.AccessCount,
				"duration":     time.Since(state.CreatedAt).String(),
			}).Info("Session removed")
		case SessionAccessed:
			if state.AccessCount%10 == 0 {
				logrus.WithFields(logrus.Fields{
					"event":        event,
					"session_id":   sessionID,
					"access_count": state.AccessCount,
					"last_access":  state.LastAccessed.Format(time.RFC3339),
				}).Debug("Session accessed")
			}
		case SessionExpired:
			logrus.WithFields(logrus.Fields{
				"event":        event,
				"session_id":   sessionID,
				"access_count": state.AccessCount,
				"duration":     time.Since(state.CreatedAt).String(),
			}).Warn("Session expired")
		}
	})

	// Add performance monitoring callback
	sessionManager.AddEventCallback(func(event SessionEvent, sessionID string, state *SessionState) {
		if event == SessionAccessed {
			if state.AccessCount > 100 {
				timeSinceCreation := time.Since(state.CreatedAt)
				accessRate := float64(state.AccessCount) / timeSinceCreation.Minutes()
				if accessRate > 10 {
					logrus.WithFields(logrus.Fields{
						"session_id":   sessionID,
						"access_rate":  accessRate,
						"access_count": state.AccessCount,
						"duration":     timeSinceCreation.String(),
					}).Warn("High frequency access pattern detected")
				}
			}
		}
	})
}

// NewSessionAwareHooks returns server.Hooks with session management
func NewSessionAwareHooks() *server.Hooks {
	hooks := &server.Hooks{}

	hooks.AddOnRegisterSession(func(ctx context.Context, sessionCli server.ClientSession) {
		sessionID := sessionCli.SessionID()
		logrus.WithField("session_id", sessionID).Info("Session registered")

		sessionManager := GetSessionManager()
		if _, err := sessionManager.GetState(sessionID); err == nil {
			sessionManager.SetSessionMetadata(sessionID, "client_connected_at", time.Now())
			sessionManager.SetSessionMetadata(sessionID, "client_type", "mcp_client")
		}
	})

	hooks.AddOnUnregisterSession(func(ctx context.Context, sessionCli server.ClientSession) {
		sessionID := sessionCli.SessionID()
		logrus.WithField("session_id", sessionID).Info("Session unregistered")

		sessionManager := GetSessionManager()
		if err := sessionManager.Remove(sessionID); err != nil {
			logrus.WithFields(logrus.Fields{
				"session_id": sessionID,
				"error":      err,
			}).Warn("Failed to cleanup session")
		}
	})

	return hooks
}
