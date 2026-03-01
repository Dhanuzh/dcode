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
	ID        string      `json:"id"`
	Title     string      `json:"title"`
	Agent     string      `json:"agent"`
	Model     string      `json:"model"`
	Provider  string      `json:"provider"`
	ParentID  string      `json:"parent_id,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
	Messages  []Message   `json:"messages"`
	Summary   *Summary    `json:"summary,omitempty"`
	Status    string      `json:"status"` // "idle", "busy", "retry"
	Revert    *RevertInfo `json:"revert,omitempty"`
}

// RevertInfo tracks the revert state for undo operations
type RevertInfo struct {
	MessageID string `json:"message_id"`
	PartID    string `json:"part_id,omitempty"`
	Snapshot  string `json:"snapshot,omitempty"` // Git tree hash for the snapshot
	Diff      string `json:"diff,omitempty"`     // Diff from snapshot to current
}

// Summary tracks session statistics
type Summary struct {
	Additions int      `json:"additions"`
	Deletions int      `json:"deletions"`
	Files     []string `json:"files"`
	FileCount int      `json:"file_count"`
	TokensIn  int      `json:"tokens_in"`
	TokensOut int      `json:"tokens_out"`
	ToolCalls int      `json:"tool_calls"`
	TotalCost float64  `json:"total_cost"`
}

// Message represents a conversation message
type Message struct {
	ID           string        `json:"id"`
	Role         string        `json:"role"` // "user", "assistant", "system"
	Content      string        `json:"content"`
	Parts        []Part        `json:"parts,omitempty"`
	CreatedAt    time.Time     `json:"created_at"`
	CompletedAt  time.Time     `json:"completed_at,omitempty"`
	TokensIn     int           `json:"tokens_in,omitempty"`
	TokensOut    int           `json:"tokens_out,omitempty"`
	TokensCache  int           `json:"tokens_cache,omitempty"`
	Cost         float64       `json:"cost,omitempty"`
	IsSummary    bool          `json:"is_summary,omitempty"`    // True for compaction summary messages
	AgentName    string        `json:"agent_name,omitempty"`    // Which agent produced this message
	ParentMsgID  string        `json:"parent_msg_id,omitempty"` // Links assistant reply to user message
	Variant      string        `json:"variant,omitempty"`       // Reasoning variant used
	ModelID      string        `json:"model_id,omitempty"`
	ProviderID   string        `json:"provider_id,omitempty"`
	FinishReason string        `json:"finish_reason,omitempty"` // "stop", "tool_use", "length", etc.
	Error        *MessageError `json:"error,omitempty"`
}

// MessageError represents an error that occurred during message processing
type MessageError struct {
	Type    string `json:"type"` // "api_error", "context_overflow", "unknown"
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// ImageAttachment holds base64-encoded image data attached to a user message.
type ImageAttachment struct {
	MediaType string `json:"media_type"`          // "image/png", "image/jpeg", etc.
	Data      string `json:"data"`                // base64-encoded bytes
	FileName  string `json:"file_name,omitempty"` // original filename for display
}

// Part represents a message part (text, tool call, reasoning, image, etc.)
type Part struct {
	Type        string                 `json:"type"` // "text", "tool_use", "tool_result", "reasoning", "image", "error", "step_start", "step_finish", "patch"
	Content     string                 `json:"content,omitempty"`
	ToolID      string                 `json:"tool_id,omitempty"`
	ToolName    string                 `json:"tool_name,omitempty"`
	ToolInput   map[string]interface{} `json:"tool_input,omitempty"`
	IsError     bool                   `json:"is_error,omitempty"`
	IsCompacted bool                   `json:"is_compacted,omitempty"` // Tool output was pruned
	IsSynthetic bool                   `json:"is_synthetic,omitempty"` // Synthetic part (e.g., auto-continue)
	Status      string                 `json:"status,omitempty"`       // "pending", "running", "completed", "error"
	Snapshot    string                 `json:"snapshot,omitempty"`     // Git tree hash for step tracking
	PatchHash   string                 `json:"patch_hash,omitempty"`   // Hash of the patch
	PatchFiles  []string               `json:"patch_files,omitempty"`  // Files changed in the patch
	StartedAt   time.Time              `json:"started_at,omitempty"`
	EndedAt     time.Time              `json:"ended_at,omitempty"`
	StepCost    float64                `json:"step_cost,omitempty"`
	StepTokens  *StepTokens            `json:"step_tokens,omitempty"`
	Title       string                 `json:"title,omitempty"`    // Tool output title
	Metadata    map[string]interface{} `json:"metadata,omitempty"` // Provider metadata
	// Image attachment (Type == "image")
	Image *ImageAttachment `json:"image,omitempty"`
}

// StepTokens tracks token usage for a single step
type StepTokens struct {
	Input      int `json:"input"`
	Output     int `json:"output"`
	Reasoning  int `json:"reasoning"`
	CacheRead  int `json:"cache_read"`
	CacheWrite int `json:"cache_write"`
}

// CostInfo tracks cost at a granular level
type CostInfo struct {
	InputCost  float64 `json:"input_cost"`
	OutputCost float64 `json:"output_cost"`
	CacheCost  float64 `json:"cache_cost"`
	Total      float64 `json:"total"`
}

// Store manages session persistence
type Store struct {
	mu        sync.RWMutex
	baseDir   string
	sessions  map[string]*Session
	statusMgr *StatusManager

	// Lazy loading: sessions are loaded in background on startup
	loadDone chan struct{}
	loadErr  error
}

// NewStore creates a new session store
func NewStore(baseDir string) (*Store, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create session directory: %w", err)
	}

	store := &Store{
		baseDir:   baseDir,
		sessions:  make(map[string]*Session),
		statusMgr: NewStatusManager(),
		loadDone:  make(chan struct{}),
	}

	// Load existing sessions in background so TUI renders immediately
	go func() {
		store.loadErr = store.loadAll()
		close(store.loadDone)
	}()

	return store, nil
}

// ensureLoaded waits for background loading to complete.
// This is called before any operation that reads from the sessions map.
func (s *Store) ensureLoaded() {
	<-s.loadDone
}

// StatusManager returns the status manager for this store
func (s *Store) StatusManager() *StatusManager {
	return s.statusMgr
}

// Create creates a new session
func (s *Store) Create(agent, model, provider string) (*Session, error) {
	s.ensureLoaded()
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
	s.ensureLoaded()
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
	s.ensureLoaded()
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
	s.ensureLoaded()
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
	session.Summary.TotalCost += msg.Cost

	return s.save(session)
}

// UpdateMessage updates an existing message in the session
func (s *Store) UpdateMessage(sessionID, messageID string, updater func(*Message)) error {
	s.ensureLoaded()
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	for i := range session.Messages {
		if session.Messages[i].ID == messageID {
			updater(&session.Messages[i])
			session.UpdatedAt = time.Now()
			return s.save(session)
		}
	}

	return fmt.Errorf("message not found: %s", messageID)
}

// ReplaceMessages replaces all messages in a session with the provided slice.
// Used by the compaction flow to swap the full history for a summary.
func (s *Store) ReplaceMessages(sessionID string, msgs []Message) error {
	s.ensureLoaded()
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Assign IDs to any messages that don't have one
	for i := range msgs {
		if msgs[i].ID == "" {
			msgs[i].ID = uuid.New().String()[:8]
		}
	}
	session.Messages = msgs
	session.UpdatedAt = time.Now()
	return s.save(session)
}

// UpdateTitle updates the session title
func (s *Store) UpdateTitle(sessionID, title string) error {
	s.ensureLoaded()
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
	s.ensureLoaded()
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.Status = status
	return nil // Don't save for status-only updates (transient)
}

// SetRevert sets the revert state for a session
func (s *Store) SetRevert(sessionID string, revert *RevertInfo) error {
	s.ensureLoaded()
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.Revert = revert
	session.UpdatedAt = time.Now()
	return s.save(session)
}

// Revert reverts a session to a specific message, using snapshots to undo file changes
func (s *Store) Revert(sessionID, messageID string, snapshot *Snapshot) error {
	s.ensureLoaded()
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Find the target message and collect patches to revert
	var patches []SnapshotPatch
	found := false
	var lastUserMsgID string

	for _, msg := range session.Messages {
		if msg.Role == "user" {
			lastUserMsgID = msg.ID
		}
		if msg.ID == messageID {
			found = true
		}
		if found {
			for _, part := range msg.Parts {
				if part.Type == "patch" && part.PatchHash != "" {
					patches = append(patches, SnapshotPatch{
						Hash:  part.PatchHash,
						Files: part.PatchFiles,
					})
				}
			}
		}
	}

	if !found {
		return fmt.Errorf("message not found: %s", messageID)
	}

	// Revert file changes using snapshot
	if snapshot != nil && len(patches) > 0 {
		if err := snapshot.Revert(patches); err != nil {
			return fmt.Errorf("failed to revert file changes: %w", err)
		}
	}

	// Track revert state
	snapshotHash := ""
	if session.Revert != nil && session.Revert.Snapshot != "" {
		snapshotHash = session.Revert.Snapshot
	} else if snapshot != nil {
		snapshotHash, _ = snapshot.Track()
	}

	session.Revert = &RevertInfo{
		MessageID: lastUserMsgID,
		Snapshot:  snapshotHash,
	}

	if snapshotHash != "" && snapshot != nil {
		diff, _ := snapshot.Diff(snapshotHash)
		session.Revert.Diff = diff
	}

	session.UpdatedAt = time.Now()
	return s.save(session)
}

// Unrevert undoes a revert, restoring the session to its pre-revert state
func (s *Store) Unrevert(sessionID string, snapshot *Snapshot) error {
	s.ensureLoaded()
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	if session.Revert == nil {
		return nil
	}

	// Restore from snapshot
	if session.Revert.Snapshot != "" && snapshot != nil {
		if err := snapshot.Restore(session.Revert.Snapshot); err != nil {
			return fmt.Errorf("failed to restore snapshot: %w", err)
		}
	}

	session.Revert = nil
	session.UpdatedAt = time.Now()
	return s.save(session)
}

// CleanupRevert removes messages after the revert point and clears the revert state
func (s *Store) CleanupRevert(sessionID string) error {
	s.ensureLoaded()
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	if session.Revert == nil {
		return nil
	}

	// Remove messages from the revert point onwards
	var preserved []Message
	for _, msg := range session.Messages {
		if msg.ID == session.Revert.MessageID {
			break
		}
		preserved = append(preserved, msg)
	}

	session.Messages = preserved
	session.Revert = nil
	session.UpdatedAt = time.Now()
	return s.save(session)
}

// Fork creates a copy of a session at a specific message point
func (s *Store) Fork(sessionID string, atMessageIdx int) (*Session, error) {
	s.ensureLoaded()
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
	s.ensureLoaded()
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
	s.ensureLoaded()
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
	s.ensureLoaded()
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

// Compact performs token-based compaction using the pruning algorithm
func (s *Store) Compact(sessionID string, keepLastN int) error {
	s.ensureLoaded()
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Use the new pruning system
	session.Messages = PruneToolOutputs(session.Messages, true)
	session.UpdatedAt = time.Now()
	return s.save(session)
}

// GetSessionCost calculates total session cost from messages
func (s *Store) GetSessionCost(sessionID string) (float64, error) {
	s.ensureLoaded()
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return 0, fmt.Errorf("session not found: %s", sessionID)
	}

	var total float64
	for _, msg := range session.Messages {
		total += msg.Cost
	}
	return total, nil
}

// GetSessionStats returns aggregate statistics for a session
func (s *Store) GetSessionStats(sessionID string) (*Summary, error) {
	s.ensureLoaded()
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	summary := &Summary{}
	for _, msg := range session.Messages {
		summary.TokensIn += msg.TokensIn
		summary.TokensOut += msg.TokensOut
		summary.TotalCost += msg.Cost
		for _, part := range msg.Parts {
			if part.Type == "tool_use" {
				summary.ToolCalls++
			}
			if part.Type == "patch" {
				for _, f := range part.PatchFiles {
					summary.Files = append(summary.Files, f)
				}
			}
		}
	}
	summary.FileCount = len(summary.Files)

	return summary, nil
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
