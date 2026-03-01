package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/Dhanuzh/dcode/internal/agent"
	"github.com/Dhanuzh/dcode/internal/config"
	"github.com/Dhanuzh/dcode/internal/provider"
	"github.com/Dhanuzh/dcode/internal/session"
	"github.com/Dhanuzh/dcode/internal/tool"
)

// Server is the HTTP API server
type Server struct {
	config     *config.Config
	store      *session.Store
	registry   *tool.Registry
	mux        *http.ServeMux
	server     *http.Server
	sseClients map[string]chan []byte
	sseMu      sync.RWMutex
}

// New creates a new API server
func New(cfg *config.Config, store *session.Store, registry *tool.Registry) *Server {
	s := &Server{
		config:     cfg,
		store:      store,
		registry:   registry,
		mux:        http.NewServeMux(),
		sseClients: make(map[string]chan []byte),
	}
	s.registerRoutes()
	return s
}

// Start starts the HTTP server
func (s *Server) Start() error {
	port := s.config.Server.Port
	if port == 0 {
		port = 4096
	}
	hostname := s.config.Server.Hostname
	if hostname == "" {
		hostname = "localhost"
	}

	addr := fmt.Sprintf("%s:%d", hostname, port)
	s.server = &http.Server{
		Addr:    addr,
		Handler: s.corsMiddleware(s.mux),
	}

	fmt.Printf("DCode API server listening on http://%s\n", addr)
	return s.server.ListenAndServe()
}

// Stop gracefully stops the server
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}

func (s *Server) registerRoutes() {
	// Health & info
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /info", s.handleInfo)

	// Sessions
	s.mux.HandleFunc("GET /session", s.handleListSessions)
	s.mux.HandleFunc("POST /session", s.handleCreateSession)
	s.mux.HandleFunc("GET /session/{id}", s.handleGetSession)
	s.mux.HandleFunc("DELETE /session/{id}", s.handleDeleteSession)
	s.mux.HandleFunc("GET /session/{id}/messages", s.handleGetMessages)
	s.mux.HandleFunc("POST /session/{id}/prompt", s.handlePrompt)
	s.mux.HandleFunc("POST /session/{id}/fork", s.handleForkSession)
	s.mux.HandleFunc("POST /session/{id}/compact", s.handleCompactSession)
	s.mux.HandleFunc("GET /session/{id}/export", s.handleExportSession)
	s.mux.HandleFunc("POST /session/import", s.handleImportSession)

	// Providers & models
	s.mux.HandleFunc("GET /provider", s.handleListProviders)
	s.mux.HandleFunc("GET /model", s.handleListModels)

	// Agents
	s.mux.HandleFunc("GET /agent", s.handleListAgents)

	// Tools
	s.mux.HandleFunc("GET /tool", s.handleListTools)

	// Config
	s.mux.HandleFunc("GET /config", s.handleGetConfig)
	s.mux.HandleFunc("PUT /config", s.handleUpdateConfig)

	// SSE Events
	s.mux.HandleFunc("GET /events", s.handleSSE)

	// Project info
	s.mux.HandleFunc("GET /project", s.handleProjectInfo)
}

// CORS middleware
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Handlers

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"status": "ok", "version": "2.0"})
}

func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"name":     "dcode",
		"version":  "2.0.0",
		"provider": s.config.Provider,
		"model":    s.config.GetDefaultModel(s.config.Provider),
		"agent":    s.config.DefaultAgent,
	})
}

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	sessions := s.store.List()
	summaries := make([]map[string]interface{}, len(sessions))
	for i, sess := range sessions {
		summaries[i] = map[string]interface{}{
			"id":         sess.ID,
			"title":      sess.Title,
			"agent":      sess.Agent,
			"model":      sess.Model,
			"status":     sess.Status,
			"messages":   len(sess.Messages),
			"created_at": sess.CreatedAt,
			"updated_at": sess.UpdatedAt,
		}
	}
	writeJSON(w, summaries)
}

func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Agent    string `json:"agent"`
		Model    string `json:"model"`
		Provider string `json:"provider"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Agent = s.config.DefaultAgent
		req.Model = s.config.GetDefaultModel(s.config.Provider)
		req.Provider = s.config.Provider
	}
	if req.Agent == "" {
		req.Agent = s.config.DefaultAgent
	}
	if req.Provider == "" {
		req.Provider = s.config.Provider
	}
	if req.Model == "" {
		req.Model = s.config.GetDefaultModel(req.Provider)
	}

	sess, err := s.store.Create(req.Agent, req.Model, req.Provider)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, sess)
}

func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sess, err := s.store.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, sess)
}

func (s *Server) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.store.Delete(id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGetMessages(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sess, err := s.store.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, sess.Messages)
}

func (s *Server) handlePrompt(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	sess, err := s.store.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// Set up SSE streaming
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "Streaming not supported")
		return
	}

	// Create provider and engine
	apiKey, err := config.GetAPIKeyWithFallback(sess.Provider, s.config)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	prov, err := createProviderFromName(sess.Provider, apiKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	ag := agent.GetAgent(sess.Agent, s.config)
	engine := session.NewPromptEngine(s.store, prov, s.config, ag, s.registry)

	engine.OnStream(func(event session.StreamEvent) {
		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	})

	ctx := r.Context()
	if err := engine.Run(ctx, id, req.Message); err != nil {
		data, _ := json.Marshal(session.StreamEvent{Type: "error", Content: err.Error()})
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	fmt.Fprintf(w, "data: {\"type\":\"done\"}\n\n")
	flusher.Flush()
}

func (s *Server) handleForkSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		AtMessage int `json:"at_message"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	forked, err := s.store.Fork(id, req.AtMessage)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, forked)
}

func (s *Server) handleCompactSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.store.Compact(id, 10); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]string{"status": "compacted"})
}

func (s *Server) handleExportSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	data, err := s.store.Export(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=dcode-session-%s.json", id))
	w.Write(data)
}

func (s *Server) handleImportSession(w http.ResponseWriter, r *http.Request) {
	data, err := readBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Failed to read body")
		return
	}
	sess, err := s.store.Import(data)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, sess)
}

func (s *Server) handleListProviders(w http.ResponseWriter, r *http.Request) {
	providers := s.config.ListAvailableProviders()
	writeJSON(w, providers)
}

func (s *Server) handleListModels(w http.ResponseWriter, r *http.Request) {
	models := config.DefaultModels
	writeJSON(w, models)
}

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	agents := agent.BuiltinAgents()
	result := make([]map[string]interface{}, 0, len(agents))
	for name, ag := range agents {
		result = append(result, map[string]interface{}{
			"name":        name,
			"description": ag.Description,
			"mode":        ag.Mode,
			"steps":       ag.Steps,
			"tools":       ag.Tools,
		})
	}
	writeJSON(w, result)
}

func (s *Server) handleListTools(w http.ResponseWriter, r *http.Request) {
	tools := s.registry.GetAll()
	result := make([]map[string]interface{}, 0, len(tools))
	for name, t := range tools {
		result = append(result, map[string]interface{}{
			"name":        name,
			"description": t.Description,
		})
	}
	writeJSON(w, result)
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	// Return safe config (no API keys)
	safe := map[string]interface{}{
		"provider":      s.config.Provider,
		"model":         s.config.GetDefaultModel(s.config.Provider),
		"default_agent": s.config.DefaultAgent,
		"streaming":     s.config.Streaming,
		"theme":         s.config.Theme,
		"auto_title":    s.config.AutoTitle,
		"snapshot":      s.config.Snapshot,
		"compaction":    s.config.Compaction,
	}
	writeJSON(w, safe)
}

func (s *Server) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	// Apply updates (limited set)
	if v, ok := updates["provider"].(string); ok {
		s.config.Provider = v
	}
	if v, ok := updates["model"].(string); ok {
		s.config.Model = v
	}
	if v, ok := updates["default_agent"].(string); ok {
		s.config.DefaultAgent = v
	}
	if v, ok := updates["theme"].(string); ok {
		s.config.Theme = v
	}

	writeJSON(w, map[string]string{"status": "updated"})
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "Streaming not supported")
		return
	}

	// Send heartbeat
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	fmt.Fprintf(w, "data: {\"type\":\"connected\"}\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

func (s *Server) handleProjectInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"directory": config.GetProjectDir(),
	})
}

// Helper functions

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func readBody(r *http.Request) ([]byte, error) {
	return io.ReadAll(r.Body)
}

// createProviderFromName creates a provider using the provider package
func createProviderFromName(name, apiKey string) (provider.Provider, error) {
	return provider.CreateProvider(name, apiKey)
}
