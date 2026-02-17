package session

import (
	"sync"
	"time"
)

// StatusType represents the session processing status
type StatusType string

const (
	StatusIdle  StatusType = "idle"
	StatusBusy  StatusType = "busy"
	StatusRetry StatusType = "retry"
)

// Status represents the current state of a session's processing
type Status struct {
	Type    StatusType `json:"type"`
	Attempt int        `json:"attempt,omitempty"` // For retry status
	Message string     `json:"message,omitempty"` // For retry status
	NextAt  time.Time  `json:"next_at,omitempty"` // For retry status
}

// StatusManager manages session statuses with thread-safe access
type StatusManager struct {
	mu       sync.RWMutex
	statuses map[string]*Status
	onChange func(sessionID string, status *Status)
}

// NewStatusManager creates a new status manager
func NewStatusManager() *StatusManager {
	return &StatusManager{
		statuses: make(map[string]*Status),
	}
}

// OnChange registers a callback for status changes
func (sm *StatusManager) OnChange(callback func(sessionID string, status *Status)) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.onChange = callback
}

// Get returns the current status for a session
func (sm *StatusManager) Get(sessionID string) *Status {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if s, ok := sm.statuses[sessionID]; ok {
		return s
	}
	return &Status{Type: StatusIdle}
}

// Set updates the status for a session
func (sm *StatusManager) Set(sessionID string, status *Status) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if status.Type == StatusIdle {
		delete(sm.statuses, sessionID)
	} else {
		sm.statuses[sessionID] = status
	}

	if sm.onChange != nil {
		sm.onChange(sessionID, status)
	}
}

// SetBusy marks a session as busy
func (sm *StatusManager) SetBusy(sessionID string) {
	sm.Set(sessionID, &Status{Type: StatusBusy})
}

// SetIdle marks a session as idle
func (sm *StatusManager) SetIdle(sessionID string) {
	sm.Set(sessionID, &Status{Type: StatusIdle})
}

// SetRetry marks a session as retrying
func (sm *StatusManager) SetRetry(sessionID string, attempt int, message string, delay time.Duration) {
	sm.Set(sessionID, &Status{
		Type:    StatusRetry,
		Attempt: attempt,
		Message: message,
		NextAt:  time.Now().Add(delay),
	})
}

// IsBusy returns true if any session is currently busy
func (sm *StatusManager) IsBusy(sessionID string) bool {
	s := sm.Get(sessionID)
	return s.Type == StatusBusy || s.Type == StatusRetry
}

// List returns all non-idle session statuses
func (sm *StatusManager) List() map[string]*Status {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make(map[string]*Status, len(sm.statuses))
	for k, v := range sm.statuses {
		result[k] = v
	}
	return result
}
