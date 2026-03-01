package session

import (
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Store CRUD
// ---------------------------------------------------------------------------

func TestStoreCreateAndGet(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	sess, err := store.Create("coder", "claude-sonnet-4-5", "anthropic")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if sess.ID == "" {
		t.Error("session ID should not be empty")
	}
	if sess.Agent != "coder" {
		t.Errorf("Agent: want coder, got %q", sess.Agent)
	}
	if sess.Provider != "anthropic" {
		t.Errorf("Provider: want anthropic, got %q", sess.Provider)
	}
	if sess.Status != "idle" {
		t.Errorf("Status: want idle, got %q", sess.Status)
	}

	got, err := store.Get(sess.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != sess.ID {
		t.Errorf("Get returned wrong session: %q vs %q", got.ID, sess.ID)
	}
}

func TestStoreGetMissing(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	_, err = store.Get("nonexistent")
	if err == nil {
		t.Error("expected error for missing session, got nil")
	}
}

func TestStoreList(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	// Create three sessions with small delays to get distinct UpdatedAt
	for i := 0; i < 3; i++ {
		if _, err := store.Create("coder", "gpt-4o", "openai"); err != nil {
			t.Fatalf("Create[%d]: %v", i, err)
		}
		time.Sleep(time.Millisecond)
	}

	sessions := store.List()
	if len(sessions) != 3 {
		t.Errorf("List: want 3 sessions, got %d", len(sessions))
	}

	// Verify newest-first ordering
	for i := 1; i < len(sessions); i++ {
		if sessions[i].UpdatedAt.After(sessions[i-1].UpdatedAt) {
			t.Errorf("List is not sorted newest-first at index %d", i)
		}
	}
}

func TestStoreDelete(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	sess, err := store.Create("coder", "claude-sonnet-4-5", "anthropic")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.Delete(sess.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := store.Get(sess.ID); err == nil {
		t.Error("expected error after Delete, got nil")
	}
}

func TestStoreAddMessage(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	sess, err := store.Create("coder", "claude-sonnet-4-5", "anthropic")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	msg := Message{
		ID:      "msg1",
		Role:    "user",
		Content: "hello",
	}
	if err := store.AddMessage(sess.ID, msg); err != nil {
		t.Fatalf("AddMessage: %v", err)
	}

	got, err := store.Get(sess.ID)
	if err != nil {
		t.Fatalf("Get after AddMessage: %v", err)
	}
	if len(got.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(got.Messages))
	}
	if got.Messages[0].Content != "hello" {
		t.Errorf("message content mismatch: %q", got.Messages[0].Content)
	}
}

func TestStoreExportImport(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	sess, err := store.Create("planner", "gpt-4o", "openai")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	_ = store.AddMessage(sess.ID, Message{ID: "m1", Role: "user", Content: "test"})

	data, err := store.Export(sess.ID)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("exported data should not be empty")
	}

	// Import into a fresh store — Import always generates a new ID to avoid
	// conflicts, so we verify the agent/model are preserved instead.
	dir2 := t.TempDir()
	store2, err := NewStore(dir2)
	if err != nil {
		t.Fatalf("NewStore2: %v", err)
	}

	imported, err := store2.Import(data)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if imported.ID == "" {
		t.Error("imported session should have a non-empty ID")
	}
	if imported.Agent != sess.Agent {
		t.Errorf("Agent mismatch: want %q, got %q", sess.Agent, imported.Agent)
	}
	if imported.Provider != sess.Provider {
		t.Errorf("Provider mismatch: want %q, got %q", sess.Provider, imported.Provider)
	}
}

// ---------------------------------------------------------------------------
// Session / Message structure
// ---------------------------------------------------------------------------

func TestMessagePartTypes(t *testing.T) {
	types := []string{"text", "tool_use", "tool_result", "reasoning", "image", "error"}
	for _, tp := range types {
		p := Part{Type: tp}
		if p.Type != tp {
			t.Errorf("Part.Type round-trip failed for %q", tp)
		}
	}
}

func TestSummaryFileCount(t *testing.T) {
	s := Summary{
		Files:     []string{"a.go", "b.go", "c.go"},
		FileCount: 3,
		TokensIn:  500,
		TokensOut: 200,
	}
	if s.FileCount != len(s.Files) {
		t.Errorf("FileCount %d != len(Files) %d", s.FileCount, len(s.Files))
	}
}

func TestRevertInfoFields(t *testing.T) {
	r := RevertInfo{
		MessageID: "msg_abc",
		PartID:    "part_xyz",
		Snapshot:  "deadbeef",
		Diff:      "--- a\n+++ b\n",
	}
	if r.MessageID == "" || r.Snapshot == "" {
		t.Error("RevertInfo fields should not be empty")
	}
}

func TestMessageErrorFields(t *testing.T) {
	e := MessageError{
		Type:    "api_error",
		Message: "rate limit hit",
		Code:    "429",
	}
	if e.Type == "" || e.Message == "" {
		t.Error("MessageError fields should not be empty")
	}
}
