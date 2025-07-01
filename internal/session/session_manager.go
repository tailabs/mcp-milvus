package session

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"github.com/sirupsen/logrus"
)

type ConnConfig struct {
	Address string `json:"address"`
	Token   string `json:"token"`
	DBName  string `json:"db_name"`
}

func (c *ConnConfig) ToMilvusClientConfig() (*milvusclient.ClientConfig, error) {
	if len(c.Token) == 0 {
		return &milvusclient.ClientConfig{
			Address: c.Address,
			DBName:  c.DBName,
		}, nil
	}

	tokenSlice := strings.Split(c.Token, ":")
	if len(tokenSlice) != 2 {
		return nil, fmt.Errorf("invalid token format, e.g. username:password")
	}

	return &milvusclient.ClientConfig{
		Address:  c.Address,
		Username: tokenSlice[0],
		Password: tokenSlice[1],
		DBName:   c.DBName,
	}, nil
}

var (
	sessionManager SessionManagerInterface
	once           sync.Once
)

// SessionEvent represents different session events
type SessionEvent string

const (
	SessionCreated  SessionEvent = "session_created"
	SessionRemoved  SessionEvent = "session_removed"
	SessionAccessed SessionEvent = "session_accessed"
	SessionExpired  SessionEvent = "session_expired"
)

// SessionState holds detailed session information
type SessionState struct {
	SessionID    string
	ConnConfig   *ConnConfig
	Client       *milvusclient.Client
	CreatedAt    time.Time
	LastAccessed time.Time
	AccessCount  int64
	Metadata     map[string]interface{}
}

// SessionEventCallback defines the callback function for session events
type SessionEventCallback func(event SessionEvent, sessionID string, state *SessionState)

// SessionManagerInterface defines the core interface for session management
// Note: This interface has been simplified to focus on essential functionality
// Some methods were removed due to Ristretto cache limitations or lack of usage
type SessionManagerInterface interface {
	// Core session operations
	Get(sessionId string) (*milvusclient.Client, error)
	GetState(sessionId string) (*SessionState, error)
	Set(sessionId string, config *ConnConfig) error
	Remove(sessionId string) error
	Clear() error
	Size() int
	Close() error

	// Event callback management
	AddEventCallback(callback SessionEventCallback)

	// Session metadata operations
	GetSessionMetadata(sessionId string) (map[string]interface{}, error)
	SetSessionMetadata(sessionId string, key string, value interface{}) error
}

// SessionManager implements the session management functionality with Ristretto cache
type SessionManager struct {
	cache     *ristretto.Cache
	callbacks []SessionEventCallback
	mu        sync.RWMutex

	// Configuration
	maxSessions int
	defaultTTL  time.Duration

	// Background cleanup
	cleanupTicker *time.Ticker
	stopChan      chan struct{}

	// Session counter (Ristretto doesn't have built-in counting)
	sessionCount int64
}

// GetSessionManager returns the global session manager instance (singleton pattern)
func GetSessionManager() SessionManagerInterface {
	once.Do(func() {
		sessionManager = NewSessionManager()
	})
	return sessionManager
}

// NewSessionManager creates a new session manager instance with Ristretto cache
func NewSessionManager() *SessionManager {
	// Create Ristretto cache configuration
	config := &ristretto.Config{
		NumCounters: 1e7,     // Number of counters, should be 10x the number of max items
		MaxCost:     1 << 30, // Maximum cost (1GB)
		BufferItems: 64,      // Buffer size
	}

	cache, err := ristretto.NewCache(config)
	if err != nil {
		logrus.Fatalf("Failed to create Ristretto cache: %v", err)
	}

	sm := &SessionManager{
		cache:        cache,
		callbacks:    make([]SessionEventCallback, 0),
		maxSessions:  100,
		defaultTTL:   1 * time.Hour,
		stopChan:     make(chan struct{}),
		sessionCount: 0,
	}

	// Start background cleanup goroutine (minimal monitoring)
	sm.startBackgroundMonitoring()

	return sm
}

// startBackgroundMonitoring starts a goroutine for basic monitoring
// Note: Ristretto handles expiration automatically, so we only log basic stats
func (s *SessionManager) startBackgroundMonitoring() {
	s.cleanupTicker = time.NewTicker(15 * time.Minute)

	go func() {
		for {
			select {
			case <-s.cleanupTicker.C:
				logrus.WithField("active_sessions", atomic.LoadInt64(&s.sessionCount)).Debug("Session manager stats")
			case <-s.stopChan:
				return
			}
		}
	}()
}

// triggerEvent fires all registered callbacks for the given event
func (s *SessionManager) triggerEvent(event SessionEvent, sessionID string, state *SessionState) {
	// Use a separate goroutine to handle event triggering to avoid blocking
	go func() {
		// Get callbacks with read lock
		s.mu.RLock()
		callbacks := make([]SessionEventCallback, len(s.callbacks))
		copy(callbacks, s.callbacks)
		s.mu.RUnlock()

		// Fire callbacks
		for _, callback := range callbacks {
			go func(cb SessionEventCallback) {
				defer func() {
					if r := recover(); r != nil {
						logrus.WithFields(logrus.Fields{
							"event":   event,
							"session": sessionID,
							"panic":   r,
						}).Error("Session event callback panicked")
					}
				}()
				cb(event, sessionID, state)
			}(callback)
		}
	}()
}

// Get retrieves the Milvus client for the specified session
func (s *SessionManager) Get(sessionId string) (*milvusclient.Client, error) {
	if sessionId == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}

	// Get session from cache
	item, found := s.cache.Get(sessionId)
	if !found {
		return nil, fmt.Errorf("session not found: %s", sessionId)
	}

	state, ok := item.(*SessionState)
	if !ok {
		return nil, fmt.Errorf("invalid session data for: %s", sessionId)
	}

	// Get the client reference
	client := state.Client

	// Update access statistics (create a copy to avoid race conditions)
	updatedState := *state
	updatedState.LastAccessed = time.Now()
	updatedState.AccessCount++

	// Update cache with new state
	s.cache.SetWithTTL(sessionId, &updatedState, 1, s.defaultTTL)

	// Trigger access event with the updated state copy
	if updatedState.Metadata != nil {
		updatedState.Metadata = make(map[string]interface{})
		for k, v := range state.Metadata {
			updatedState.Metadata[k] = v
		}
	}
	s.triggerEvent(SessionAccessed, sessionId, &updatedState)

	return client, nil
}

// GetState retrieves the complete session state
func (s *SessionManager) GetState(sessionId string) (*SessionState, error) {
	if sessionId == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}

	item, found := s.cache.Get(sessionId)
	if !found {
		return nil, fmt.Errorf("session not found: %s", sessionId)
	}

	state, ok := item.(*SessionState)
	if !ok {
		return nil, fmt.Errorf("invalid session data for: %s", sessionId)
	}

	// Return a copy to prevent external modification
	stateCopy := *state
	stateCopy.Metadata = make(map[string]interface{})
	for k, v := range state.Metadata {
		stateCopy.Metadata[k] = v
	}

	return &stateCopy, nil
}

// Set creates or updates a Milvus client for the specified session
func (s *SessionManager) Set(sessionId string, config *ConnConfig) error {
	if sessionId == "" {
		return fmt.Errorf("session ID cannot be empty")
	}
	if config == nil {
		return fmt.Errorf("connection config cannot be nil")
	}

	// Check session limit
	currentCount := atomic.LoadInt64(&s.sessionCount)
	if currentCount >= int64(s.maxSessions) {
		return fmt.Errorf("maximum number of sessions (%d) reached", s.maxSessions)
	}

	// Clean up existing session if it exists
	if existing, found := s.cache.Get(sessionId); found {
		if existingState, ok := existing.(*SessionState); ok {
			s.closeClientSafely(existingState.Client, sessionId)
		}
	}

	// Create new Milvus client
	milvusClientConfig, err := config.ToMilvusClientConfig()
	if err != nil {
		return fmt.Errorf("failed to parse milvus config: %w", err)
	}

	// RetryInterceptor not flexible
	// issue:https://github.com/milvus-io/milvus/issues/42949
	client, err := milvusclient.New(context.TODO(), milvusClientConfig)
	if err != nil {
		return fmt.Errorf("failed to create milvus client: %w", err)
	}

	// Create session state
	now := time.Now()
	state := &SessionState{
		SessionID:    sessionId,
		ConnConfig:   config,
		Client:       client,
		CreatedAt:    now,
		LastAccessed: now,
		AccessCount:  0,
		Metadata:     make(map[string]interface{}),
	}

	// Store in cache
	s.cache.SetWithTTL(sessionId, state, 1, s.defaultTTL)
	atomic.AddInt64(&s.sessionCount, 1)

	// Trigger creation event
	s.triggerEvent(SessionCreated, sessionId, state)

	logrus.WithFields(logrus.Fields{
		"session":        sessionId,
		"address":        config.Address,
		"database":       config.DBName,
		"total_sessions": atomic.LoadInt64(&s.sessionCount),
	}).Info("Session created successfully")

	return nil
}

// Remove removes the specified session and cleans up resources
func (s *SessionManager) Remove(sessionId string) error {
	if sessionId == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	item, found := s.cache.Get(sessionId)
	if !found {
		return fmt.Errorf("session not found: %s", sessionId)
	}

	state, ok := item.(*SessionState)
	if !ok {
		return fmt.Errorf("invalid session data for: %s", sessionId)
	}

	// Close client safely
	s.closeClientSafely(state.Client, sessionId)

	// Remove from cache
	s.cache.Del(sessionId)
	atomic.AddInt64(&s.sessionCount, -1)

	// Trigger removal event
	s.triggerEvent(SessionRemoved, sessionId, state)

	logrus.WithField("session", sessionId).Info("Session removed successfully")
	return nil
}

// closeClientSafely closes a Milvus client with error handling
func (s *SessionManager) closeClientSafely(client *milvusclient.Client, sessionId string) {
	if client != nil {
		if err := client.Close(context.Background()); err != nil {
			logrus.WithFields(logrus.Fields{
				"session": sessionId,
				"error":   err,
			}).Warn("Failed to close milvus client for session")
		}
	}
}

// Clear removes all sessions and cleans up all resources
func (s *SessionManager) Clear() error {
	// This is a simplified approach since Ristretto doesn't provide iteration
	s.cache.Clear()
	atomic.StoreInt64(&s.sessionCount, 0)

	logrus.Info("All sessions cleared")
	return nil
}

// Size returns the current number of sessions
func (s *SessionManager) Size() int {
	return int(atomic.LoadInt64(&s.sessionCount))
}

// GetSessionMetadata retrieves metadata for a session
func (s *SessionManager) GetSessionMetadata(sessionId string) (map[string]interface{}, error) {
	state, err := s.GetState(sessionId)
	if err != nil {
		return nil, err
	}

	// Return a copy
	metadata := make(map[string]interface{})
	for k, v := range state.Metadata {
		metadata[k] = v
	}
	return metadata, nil
}

// SetSessionMetadata sets metadata for a session
func (s *SessionManager) SetSessionMetadata(sessionId string, key string, value interface{}) error {
	item, found := s.cache.Get(sessionId)
	if !found {
		return fmt.Errorf("session not found: %s", sessionId)
	}

	state, ok := item.(*SessionState)
	if !ok {
		return fmt.Errorf("invalid session data for: %s", sessionId)
	}

	// Create a copy and update metadata
	updatedState := *state
	if updatedState.Metadata == nil {
		updatedState.Metadata = make(map[string]interface{})
	} else {
		// Deep copy metadata
		updatedState.Metadata = make(map[string]interface{})
		for k, v := range state.Metadata {
			updatedState.Metadata[k] = v
		}
	}
	updatedState.Metadata[key] = value

	// Update cache
	s.cache.SetWithTTL(sessionId, &updatedState, 1, s.defaultTTL)
	return nil
}

// AddEventCallback adds a callback for session events
func (s *SessionManager) AddEventCallback(callback SessionEventCallback) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.callbacks = append(s.callbacks, callback)
}

// Close closes the session manager and cleans up all resources
func (s *SessionManager) Close() error {
	// Stop background monitoring
	if s.cleanupTicker != nil {
		s.cleanupTicker.Stop()
	}

	close(s.stopChan)

	// Clear all sessions
	s.Clear()

	// Close the cache
	s.cache.Close()

	return nil
}
