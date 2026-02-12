package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Session represents a conversation session
type Session struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Agent     string    `json:"agent"`
	Model     string    `json:"model"`
	Provider  string    `json:"provider"`
	ParentID  string    `json:"parent_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Messages  []Message `json:"messages"`
	Summary   *Summary  `json:"summary,omitempty"`
	Status    string    `json:"status"` // "idle", "busy", "retry"
}

// Summary tracks session statistics
type Summary struct {
	Additions  int      `json:"additions"`
	Deletions  int      `json:"deletions"`
	Files      []string `json:"files"`
	TokensIn   int      `json:"tokens_in"`
	TokensOut  int      `json:"tokens_out"`
	ToolCalls  int      `json:"tool_calls"`
	TotalCost  float64  `json:"total_cost"`
}

// Message represents a conversation message
type Message struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"` // "user", "assistant", "system"
	Content   string    `json:"content"`
	Parts     []Part    `json:"parts,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	TokensIn  int       `json:"tokens_in,omitempty"`
	TokensOut int       `json:"tokens_out,omitempty"`
}

// Part represents a message part (text, tool call, reasoning, etc.)
type Part struct {
	Type      string                 `json:"type"` // "text", "tool_use", "tool_result", "reasoning", "error"
	Content   string                 `json:"content,omitempty"`
	ToolID    string                 `json:"tool_id,omitempty"`
	ToolName  string                 `json:"tool_name,omitempty"`
	ToolInput map[string]interface{} `json:"tool_input,omitempty"`
	IsError   bool                   `json:"is_error,omitempty"`
	Status    string                 `json:"status,omitempty"` // "pending", "running", "completed", "error"
}

// Store manages session persistence
type Store struct {
	mu       sync.RWMutex
	baseDir  string
	sessions map[string]*Session
}

// NewStore creates a new session store
func NewStore(baseDir string) (*Store, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create session directory: %w", err)
	}

	store := &Store{
		baseDir:  baseDir,
		sessions: make(map[string]*Session),
	}

	// Load existing sessions
	if err := store.loadAll(); err != nil {
		return nil, err
	}

	return store, nil
}

// Create creates a new session
func (s *Store) Create(agent, model, provider string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session := &Session{
		ID:        uuid.New().String()[:8],
		Title:     "New Session",
		Agent:     agent,
		Model:     model,
		Provider:  provider,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Messages:  []Message{},
		Summary:   &Summary{},
		Status:    "idle",
	}

	s.sessions[session.ID] = session

	if err := s.save(session); err != nil {
		return nil, err
	}

	return session, nil
}

// Get retrieves a session by ID
func (s *Store) Get(id string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", id)
	}
	return session, nil
}

// List returns all sessions sorted by updated time (newest first)
func (s *Store) List() []*Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessions := make([]*Session, 0, len(s.sessions))
	for _, sess := range s.sessions {
		sessions = append(sessions, sess)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	return sessions
}

// AddMessage adds a message to a session
func (s *Store) AddMessage(sessionID string, msg Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	if msg.ID == "" {
		msg.ID = uuid.New().String()[:8]
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}

	session.Messages = append(session.Messages, msg)
	session.UpdatedAt = time.Now()

	// Update token summary
	if session.Summary == nil {
		session.Summary = &Summary{}
	}
	session.Summary.TokensIn += msg.TokensIn
	session.Summary.TokensOut += msg.TokensOut

	return s.save(session)
}

// UpdateTitle updates the session title
func (s *Store) UpdateTitle(sessionID, title string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.Title = title
	session.UpdatedAt = time.Now()
	return s.save(session)
}

// UpdateStatus updates the session status
func (s *Store) UpdateStatus(sessionID, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.Status = status
	return nil // Don't save for status-only updates (transient)
}

// Fork creates a copy of a session at a specific message point
func (s *Store) Fork(sessionID string, atMessageIdx int) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	original, ok := s.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	forked := &Session{
		ID:        uuid.New().String()[:8],
		Title:     original.Title + " (fork)",
		Agent:     original.Agent,
		Model:     original.Model,
		Provider:  original.Provider,
		ParentID:  original.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Summary:   &Summary{},
		Status:    "idle",
	}

	// Copy messages up to the specified index
	end := atMessageIdx
	if end > len(original.Messages) || end <= 0 {
		end = len(original.Messages)
	}

	forked.Messages = make([]Message, end)
	for i := 0; i < end; i++ {
		msg := original.Messages[i]
		msg.ID = uuid.New().String()[:8] // New IDs
		forked.Messages[i] = msg
	}

	s.sessions[forked.ID] = forked

	if err := s.save(forked); err != nil {
		return nil, err
	}

	return forked, nil
}

// Delete removes a session
func (s *Store) Delete(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sessions[sessionID]; !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	delete(s.sessions, sessionID)

	path := s.sessionPath(sessionID)
	return os.Remove(path)
}

// Export returns session data as JSON
func (s *Store) Export(sessionID string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	return json.MarshalIndent(session, "", "  ")
}

// Import loads a session from JSON data
func (s *Store) Import(data []byte) (*Session, error) {
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("invalid session data: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate new ID to avoid conflicts
	session.ID = uuid.New().String()[:8]
	session.UpdatedAt = time.Now()

	s.sessions[session.ID] = &session

	if err := s.save(&session); err != nil {
		return nil, err
	}

	return &session, nil
}

// GetLatest returns the most recently updated session
func (s *Store) GetLatest() *Session {
	sessions := s.List()
	if len(sessions) == 0 {
		return nil
	}
	return sessions[0]
}

// Compact removes old tool result content from messages to save context space
func (s *Store) Compact(sessionID string, keepLastN int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	if keepLastN <= 0 {
		keepLastN = 10 // Keep last 10 messages uncompacted
	}

	cutoff := len(session.Messages) - keepLastN
	if cutoff <= 0 {
		return nil
	}

	for i := 0; i < cutoff; i++ {
		msg := &session.Messages[i]
		for j := range msg.Parts {
			if msg.Parts[j].Type == "tool_result" && len(msg.Parts[j].Content) > 200 {
				msg.Parts[j].Content = msg.Parts[j].Content[:200] + "\n... (compacted)"
			}
		}
	}

	session.UpdatedAt = time.Now()
	return s.save(session)
}

// Internal helpers

func (s *Store) sessionPath(id string) string {
	return filepath.Join(s.baseDir, id+".json")
}

func (s *Store) save(session *Session) error {
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	path := s.sessionPath(session.ID)
	return os.WriteFile(path, data, 0644)
}

func (s *Store) loadAll() error {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		path := filepath.Join(s.baseDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var session Session
		if err := json.Unmarshal(data, &session); err != nil {
			continue
		}

		s.sessions[session.ID] = &session
	}

	return nil
}
