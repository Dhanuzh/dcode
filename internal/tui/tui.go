package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Dhanuzh/dcode/internal/agent"
	"github.com/Dhanuzh/dcode/internal/config"
	"github.com/Dhanuzh/dcode/internal/provider"
	"github.com/Dhanuzh/dcode/internal/session"
	"github.com/Dhanuzh/dcode/internal/theme"
	"github.com/Dhanuzh/dcode/internal/tool"
	"github.com/Dhanuzh/dcode/internal/tui/components"
)

// ─── Views ──────────────────────────────────────────────────────────────────────

type View string

const (
	ViewChat           View = "chat"
	ViewSessions       View = "sessions"
	ViewHelp           View = "help"
	ViewProviders      View = "providers"
	ViewModels         View = "models"
	ViewAgents         View = "agents"
	ViewCommandPalette View = "command_palette"
	ViewSettings       View = "settings"
)

// ─── Styles (Catppuccin Mocha) ──────────────────────────────────────────────────

var (
	purple  = lipgloss.Color("#CBA6F7")
	blue    = lipgloss.Color("#89B4FA")
	green   = lipgloss.Color("#A6E3A1")
	red     = lipgloss.Color("#F38BA8")
	yellow  = lipgloss.Color("#F9E2AF")
	txtClr  = lipgloss.Color("#CDD6F4")
	subtext = lipgloss.Color("#A6ADC8")
	overlay = lipgloss.Color("#6C7086")
	surface = lipgloss.Color("#313244")
	base    = lipgloss.Color("#1E1E2E")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(purple).
			Background(base).
			Padding(0, 1)

	userMsgStyle = lipgloss.NewStyle().
			Foreground(blue).
			Bold(true)

	assistantMsgStyle = lipgloss.NewStyle().
				Foreground(txtClr)

	toolCallStyle = lipgloss.NewStyle().
			Foreground(yellow).
			Italic(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(red)

	successStyle = lipgloss.NewStyle().
			Foreground(green)

	dimStyle = lipgloss.NewStyle().
			Foreground(overlay)

	highlightStyle = lipgloss.NewStyle().
			Foreground(purple)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(base).
				Background(purple).
				Bold(true).
				Padding(0, 1)

	unselectedItemStyle = lipgloss.NewStyle().
				Foreground(txtClr).
				Padding(0, 1)

	dialogTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(purple).
				Padding(0, 1)

	dialogBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(purple).
			Padding(1, 2)

	agentBadge = lipgloss.NewStyle().
			Foreground(base).
			Background(purple).
			Padding(0, 1).
			Bold(true)

	modelBadge = lipgloss.NewStyle().
			Foreground(base).
			Background(blue).
			Padding(0, 1)

	providerBadge = lipgloss.NewStyle().
			Foreground(base).
			Background(green).
			Padding(0, 1)

	keybindStyle = lipgloss.NewStyle().
			Foreground(purple).
			Bold(true)

	descStyle = lipgloss.NewStyle().
			Foreground(subtext)
)

// ─── Messages ───────────────────────────────────────────────────────────────────

// StreamMsg carries streaming content
type StreamMsg struct {
	Event session.StreamEvent
}

// DoneMsg indicates streaming is complete
type DoneMsg struct{}

// ErrorMsg carries errors
type ErrorMsg struct {
	Error error
}

// SessionCreatedMsg is sent when a new session is created
type SessionCreatedMsg struct {
	Session *session.Session
}

// ProviderChangedMsg is sent when the provider/model changes
type ProviderChangedMsg struct {
	Provider string
	Model    string
}

// ─── Streaming display types ────────────────────────────────────────────────────

// streamingToolCall tracks a tool call during streaming
type streamingToolCall struct {
	Name   string
	Detail string
	Active bool
}

// retryDisplayInfo tracks retry state during streaming
type retryDisplayInfo struct {
	Attempt int
	Message string
	NextAt  time.Time
}

// ─── Types for dialogs ──────────────────────────────────────────────────────────

// ProviderInfo holds display data for the provider selection dialog
type ProviderInfo struct {
	Name        string
	DisplayName string
	Description string
	Connected   bool
}

// ModelItem holds display data for the model selection dialog
type ModelItem struct {
	ID       string
	Name     string
	Provider string
	Context  int
	Selected bool
}

// Command defines an entry in the command palette
type Command struct {
	ID       string
	Title    string
	Category string
	Keybind  string
	Slash    string
}

func allCommands() []Command {
	return []Command{
		{ID: "model.choose", Title: "Select Model", Category: "Model", Keybind: "Ctrl+K", Slash: "/model"},
		{ID: "provider.connect", Title: "Connect Provider", Category: "Provider", Keybind: "Ctrl+P", Slash: "/provider"},
		{ID: "agent.cycle", Title: "Cycle Agent (Tab/Shift+Tab)", Category: "Agent", Keybind: "Tab", Slash: "/agent"},
		{ID: "session.new", Title: "New Session", Category: "Session", Keybind: "Ctrl+N", Slash: "/new"},
		{ID: "session.list", Title: "List Sessions", Category: "Session", Keybind: "Ctrl+L"},
		{ID: "theme.change", Title: "Change Theme", Category: "General", Slash: "/theme"},
		{ID: "settings.open", Title: "Settings", Category: "General", Keybind: "Ctrl+S"},
		{ID: "help", Title: "Help", Category: "General", Slash: "/help"},
		{ID: "compact", Title: "Compact Session", Category: "Session", Slash: "/compact"},
		{ID: "export", Title: "Export Session", Category: "Session", Slash: "/export"},
		{ID: "todo", Title: "Show Todos", Category: "General", Slash: "/todo"},
		{ID: "clear", Title: "Clear Screen", Category: "General", Slash: "/clear"},
		{ID: "quit", Title: "Quit", Category: "General", Keybind: "Ctrl+C", Slash: "/quit"},
	}
}

// ─── Model ──────────────────────────────────────────────────────────────────────

// Model is the main TUI model
type Model struct {
	// UI components
	viewport viewport.Model
	textarea textarea.Model
	spinner  spinner.Model

	// Enhanced TUI components
	syntaxHighlighter *components.SyntaxHighlighter
	markdownRenderer  *components.MarkdownRenderer
	diffViewer        *components.DiffViewer
	themeRegistry     *theme.Registry
	currentTheme      *theme.Theme

	// State
	view          View
	previousView  View
	width         int
	height        int
	sessionID     string
	messages      []session.Message
	streamingText     *strings.Builder
	streamingThinking *strings.Builder
	streamingTools    []streamingToolCall
	retryInfo         *retryDisplayInfo
	isStreaming        bool
	currentTool        string
	statusMsg     string
	statusExpiry  time.Time
	focusInput    bool // true = textarea focused, false = viewport focused

	// Provider initialization state
	providerInitializing bool
	providerInitError    error

	// Dialog state
	dialogSelected  int
	dialogFilter    string
	sessionList     []*session.Session
	selectedSession int

	// Provider/model state
	providerList []ProviderInfo
	modelList    []ModelItem
	commandList  []Command
	filteredCmds []Command

	// Dependencies
	Store    *session.Store
	Engine   *session.PromptEngine
	Config   *config.Config
	Agent    string
	Model_   string
	Provider string
	Todos    []tool.TodoItem

	// Streaming
	streamCh chan tea.Msg
	cancel   context.CancelFunc

	// Token tracking and loading states
	tokenTracker *TokenUsageTracker
	loadingState LoadingState
}

// New creates a new TUI model (opencode-style)
func New(store *session.Store, engine *session.PromptEngine, cfg *config.Config, agentName, modelName, prov string) Model {
	ta := textarea.New()
	ta.Placeholder = "Message DCode... (Enter to send, / for commands)"
	ta.Focus()
	ta.CharLimit = 50000
	ta.SetHeight(3)
	ta.ShowLineNumbers = false

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(purple)

	vp := viewport.New(80, 20)
	vp.SetContent("")

	// Initialize theme system
	themeRegistry := theme.NewRegistry()
	themeName := "catppuccin-mocha" // default
	if cfg.Theme != "" {
		themeName = cfg.Theme
	}
	_ = themeRegistry.SetCurrent(themeName)
	currentTheme := themeRegistry.Current()

	// Initialize enhanced components with theme
	syntaxHighlighter := components.NewSyntaxHighlighter(currentTheme.SyntaxTheme)
	markdownRenderer, _ := components.NewMarkdownRenderer(80, currentTheme.MarkdownTheme)
	diffViewer := components.NewDiffViewer(100, currentTheme.SyntaxTheme)

	// Get context window size for token tracking
	maxTokens := 200000 // Default
	if cfg.MaxTokens > 0 {
		maxTokens = cfg.MaxTokens
	}
	tokenTracker := NewTokenUsageTracker(maxTokens)

	return Model{
		viewport:          vp,
		textarea:          ta,
		spinner:           sp,
		view:              ViewChat,
		focusInput:        true,
		streamingText:      &strings.Builder{},
		streamingThinking:  &strings.Builder{},
		streamingTools:     nil,
		retryInfo:          nil,
		Agent:             agentName,
		Model_:            modelName,
		Provider:          prov,
		Store:             store,
		Engine:            engine,
		Config:            cfg,
		syntaxHighlighter: syntaxHighlighter,
		markdownRenderer:  markdownRenderer,
		diffViewer:        diffViewer,
		themeRegistry:     themeRegistry,
		currentTheme:      currentTheme,
		tokenTracker:      tokenTracker,
		loadingState:      LoadingState{IsActive: false},
	}
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		textarea.Blink,
		m.spinner.Tick,
		tea.EnableMouseAllMotion, // Enable mouse support for scrolling and selection
	}
	// Initialize provider asynchronously so the TUI renders immediately
	if m.Engine == nil {
		cmds = append(cmds, m.initEngineAsync())
	}
	return tea.Batch(cmds...)
}

// ProviderInitStartMsg signals that async provider initialization has started
type ProviderInitStartMsg struct{}

// initEngineAsync starts provider init and first sends a message to set the flag on the real model
func (m *Model) initEngineAsync() tea.Cmd {
	return tea.Batch(
		func() tea.Msg { return ProviderInitStartMsg{} },
		m.reinitEngine(),
	)
}

// ═══════════════════════════════════════════════════════════════════════════════
// UPDATE
// ═══════════════════════════════════════════════════════════════════════════════

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 8
		m.textarea.SetWidth(msg.Width - 4)

		// Update component widths
		if m.markdownRenderer != nil {
			m.markdownRenderer.SetWidth(msg.Width - 4)
		}

		// Keep spinner ticking if provider is still initializing
		if m.providerInitializing {
			return m, m.spinner.Tick
		}
		return m, nil

	case tea.MouseMsg:
		// Handle mouse events
		switch msg.Type {
		case tea.MouseWheelUp:
			// Scroll up in viewport
			if m.view == ViewChat && !m.focusInput {
				m.viewport.LineUp(3)
			}
		case tea.MouseWheelDown:
			// Scroll down in viewport
			if m.view == ViewChat && !m.focusInput {
				m.viewport.LineDown(3)
			}
		case tea.MouseLeft:
			// Click to focus
			if m.view == ViewChat {
				// Determine if click is in viewport or textarea area
				// Textarea is at the bottom (height - 3 to height)
				if msg.Y < m.height-4 {
					// Click in viewport area - focus viewport
					m.focusInput = false
					m.textarea.Blur()
				} else {
					// Click in textarea area - focus textarea
					m.focusInput = true
					m.textarea.Focus()
				}
			}
		}
		return m, nil

	case tea.KeyMsg:
		// Global quit
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		// Global help shortcut
		if msg.String() == "?" {
			m.previousView = m.view
			m.view = ViewHelp
			m.blurTextarea()
			return m, nil
		}
		// Dispatch to current view
		switch m.view {
		case ViewChat:
			return m.updateChat(msg)
		case ViewProviders:
			return m.updateDialog(msg, m.providerDialogLen(), m.onProviderSelect)
		case ViewModels:
			return m.updateModelDialog(msg)
		case ViewAgents:
			return m.updateDialog(msg, len(agentNames()), m.onAgentSelect)
		case ViewSessions:
			return m.updateSessionList(msg)
		case ViewCommandPalette:
			return m.updateCommandPalette(msg)
		case ViewSettings:
			return m.updateDialog(msg, len(m.settingsItems()), m.onSettingsSelect)
		case ViewHelp:
			if msg.String() == "esc" || msg.String() == "q" {
				m.view = m.previousView
				m.focusTextarea()
			}
			return m, nil
		}

	case StreamMsg:
		return m.handleStreamMsg(msg)

	case DoneMsg:
		return m.handleDoneMsg()

	case SessionCreatedMsg:
		m.sessionID = msg.Session.ID
		m.messages = []session.Message{}
		m.setStatus("Session created")
		return m, nil

	case ErrorMsg:
		m.isStreaming = false
		m.streamingThinking.Reset()
		m.streamingTools = nil
		m.retryInfo = nil
		// If this is a provider init error, store it and make it permanent
		if m.providerInitializing {
			m.providerInitializing = false
			m.providerInitError = msg.Error
			m.statusMsg = "Error: " + msg.Error.Error()
			m.statusExpiry = time.Time{} // Never expire
		} else {
			m.setStatus("Error: " + msg.Error.Error())
		}
		return m, nil

	case ProviderInitStartMsg:
		m.providerInitializing = true
		return m, nil

	case ProviderChangedMsg:
		m.providerInitializing = false
		m.providerInitError = nil
		m.Provider = msg.Provider
		m.Model_ = msg.Model
		m.setStatus(fmt.Sprintf("Switched to %s / %s", msg.Provider, msg.Model))
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update textarea (only in chat, not streaming)
	if m.view == ViewChat && !m.isStreaming {
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update viewport
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// ─── Chat key handler ───────────────────────────────────────────────────────────

func (m *Model) focusTextarea() {
	m.focusInput = true
	m.textarea.Focus()
}

func (m *Model) blurTextarea() {
	m.focusInput = false
	m.textarea.Blur()
}

func (m Model) updateChat(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		if m.focusInput {
			// If input focused, toggle to viewport
			m.blurTextarea()
		} else {
			// If viewport focused, toggle to input
			m.focusTextarea()
		}
		return m, nil
	case "shift+tab":
		if m.focusInput {
			m.blurTextarea()
		} else {
			m.focusTextarea()
		}
		return m, nil
	case "ctrl+y":
		// Copy last assistant message to clipboard
		return m.copyLastMessage()
	case "ctrl+k":
		m.blurTextarea()
		return m.openModelDialog()
	case "ctrl+j":
		return m.cycleAgent(1)
	case "ctrl+shift+p":
		m.blurTextarea()
		return m.openCommandPalette()
	case "ctrl+s":
		m.blurTextarea()
		return m.openSettings()
	case "ctrl+n":
		return m, m.createSession()
	case "ctrl+l":
		m.previousView = m.view
		m.view = ViewSessions
		m.sessionList = m.Store.List()
		m.selectedSession = 0
		return m, nil
	case "ctrl+shift+l":
		// Clear screen (messages)
		m.messages = []session.Message{}
		if m.tokenTracker != nil {
			m.tokenTracker.Reset()
		}
		m.updateViewport()
		m.setStatus("Screen cleared")
		return m, nil
	case "ctrl+p":
		m.blurTextarea()
		return m.openProviderDialog()
	case "enter":
		if m.isStreaming {
			return m, nil
		}
		// If viewport focused, switch to input
		if !m.focusInput {
			m.focusTextarea()
			return m, nil
		}
		input := strings.TrimSpace(m.textarea.Value())
		if input == "" {
			return m, nil
		}
		if strings.HasPrefix(input, "/") {
			m.textarea.Reset()
			return m.handleSlashCommand(input)
		}
		m.textarea.Reset()
		return m, m.sendMessage(input)
	case "alt+enter":
		m.textarea.InsertString("\n")
		return m, nil
	case "esc":
		if m.isStreaming {
			if m.cancel != nil {
				m.cancel()
				m.cancel = nil
			}
			m.isStreaming = false
			m.streamingText.Reset()
			m.streamingThinking.Reset()
			m.streamingTools = nil
			m.retryInfo = nil
			m.streamCh = nil
			m.setStatus("Cancelled")
			if m.sessionID != "" {
				if sess, err := m.Store.Get(m.sessionID); err == nil {
					m.messages = sess.Messages
				}
			}
			m.updateViewport()
			return m, nil
		}
	}

	// If viewport is focused, let it handle scroll keys
	if !m.focusInput {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	// Unhandled key → forward to textarea for normal typing
	if !m.isStreaming {
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd
	}
	return m, nil
}

// copyLastMessage copies the last assistant message to clipboard
func (m Model) copyLastMessage() (tea.Model, tea.Cmd) {
	// Find the last assistant message
	var lastAssistantMsg *session.Message
	for i := len(m.messages) - 1; i >= 0; i-- {
		if m.messages[i].Role == "assistant" {
			lastAssistantMsg = &m.messages[i]
			break
		}
	}
	
	if lastAssistantMsg == nil {
		m.setStatus("No assistant message to copy")
		return m, nil
	}
	
	// Extract text content from message
	content := lastAssistantMsg.Content
	
	// If content is empty, try to get from parts
	if content == "" && len(lastAssistantMsg.Parts) > 0 {
		var textParts []string
		for _, part := range lastAssistantMsg.Parts {
			if part.Type == "text" && part.Content != "" {
				textParts = append(textParts, part.Content)
			}
		}
		content = strings.Join(textParts, "\n")
	}
	
	if content == "" {
		m.setStatus("No text content to copy")
		return m, nil
	}
	
	// Copy to clipboard
	err := clipboard.WriteAll(content)
	if err != nil {
		m.setStatus("Failed to copy: " + err.Error())
		return m, nil
	}
	
	// Show success message with character count
	charCount := len(content)
	m.setStatus(fmt.Sprintf("Copied %d characters to clipboard", charCount))
	return m, nil
}

// ─── Generic dialog updater (up/down/esc/enter) ────────────────────────────────

func (m Model) updateDialog(msg tea.KeyMsg, itemCount int, onSelect func() (tea.Model, tea.Cmd)) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.view = m.previousView
		m.focusTextarea()
		return m, nil
	case "up", "k":
		if m.dialogSelected > 0 {
			m.dialogSelected--
		}
		return m, nil
	case "down", "j":
		if m.dialogSelected < itemCount-1 {
			m.dialogSelected++
		}
		return m, nil
	case "enter":
		return onSelect()
	}
	return m, nil
}

// ─── Provider Dialog ────────────────────────────────────────────────────────────

func (m Model) openProviderDialog() (tea.Model, tea.Cmd) {
	m.previousView = m.view
	m.view = ViewProviders
	m.dialogSelected = 0
	m.providerList = m.buildProviderList()
	return m, nil
}

func (m *Model) buildProviderList() []ProviderInfo {
	available := m.Config.ListAvailableProviders()
	set := make(map[string]bool)
	for _, p := range available {
		set[p] = true
	}
	all := []ProviderInfo{
		{Name: "anthropic", DisplayName: "Anthropic", Description: "Claude Opus 4.6, Sonnet 4.5, Opus 4.5, Haiku 4.5"},
		{Name: "openai", DisplayName: "OpenAI", Description: "GPT-5.2, GPT-5.1, o3, o4-mini"},
		{Name: "copilot", DisplayName: "GitHub Copilot", Description: "Claude, GPT-5, Gemini via GitHub"},
		{Name: "google", DisplayName: "Google Gemini", Description: "Gemini 3 Flash/Pro, 2.5 Pro/Flash"},
		{Name: "xai", DisplayName: "xAI", Description: "Grok 4, Grok 3, Grok Code"},
		{Name: "groq", DisplayName: "Groq", Description: "Llama, Qwen, Kimi (ultra-fast)"},
		{Name: "openrouter", DisplayName: "OpenRouter", Description: "Multi-provider gateway"},
		{Name: "deepseek", DisplayName: "DeepSeek", Description: "DeepSeek Chat, Reasoner"},
		{Name: "mistral", DisplayName: "Mistral", Description: "Devstral, Magistral, Codestral"},
		{Name: "bedrock", DisplayName: "Amazon Bedrock", Description: "Claude, Llama via AWS"},
		{Name: "deepinfra", DisplayName: "DeepInfra", Description: "GLM, Kimi, DeepSeek, GPT-OSS"},
		{Name: "cerebras", DisplayName: "Cerebras", Description: "Qwen, GPT-OSS, GLM (fast)"},
		{Name: "google-vertex", DisplayName: "Google Vertex AI", Description: "Gemini models via GCP"},
		{Name: "gitlab", DisplayName: "GitLab Duo", Description: "Claude, GPT via GitLab"},
		{Name: "cloudflare-workers-ai", DisplayName: "Cloudflare Workers AI", Description: "GPT-OSS, Llama, Qwen on Edge"},
		{Name: "together", DisplayName: "Together AI", Description: "Open models (fast inference)"},
		{Name: "azure", DisplayName: "Azure OpenAI", Description: "OpenAI models via Azure"},
		{Name: "sambanova", DisplayName: "SambaNova", Description: "Llama, DeepSeek (fast)"},
		{Name: "fireworks", DisplayName: "Fireworks AI", Description: "Llama, Qwen, Mixtral"},
		{Name: "huggingface", DisplayName: "Hugging Face", Description: "Open models via Inference API"},
	}
	for i := range all {
		all[i].Connected = set[all[i].Name]
	}
	return all
}

func (m *Model) providerDialogLen() int { return len(m.providerList) }

func (m Model) onProviderSelect() (tea.Model, tea.Cmd) {
	if m.dialogSelected >= len(m.providerList) {
		return m, nil
	}
	sel := m.providerList[m.dialogSelected]
	m.Provider = sel.Name
	m.Model_ = m.Config.GetDefaultModel(sel.Name)
	m.view = m.previousView
	m.focusTextarea()
	// Show connection status in the status message
	connStatus := ""
	if sel.Connected {
		connStatus = " (connected)"
	} else {
		connStatus = " (not connected)"
	}
	m.setStatus("Provider: " + sel.DisplayName + connStatus)
	return m, nil
}

// ─── Model Dialog ───────────────────────────────────────────────────────────────

func (m Model) openModelDialog() (tea.Model, tea.Cmd) {
	m.previousView = m.view
	m.view = ViewModels
	m.dialogSelected = 0
	m.dialogFilter = ""
	m.modelList = m.buildModelList()
	return m, nil
}

func (m *Model) buildModelList() []ModelItem {
	var models []ModelItem
	available := m.Config.ListAvailableProviders()
	for _, provName := range available {
		apiKey, _ := config.GetAPIKeyWithFallback(provName, m.Config)
		prov, err := provider.CreateProvider(provName, apiKey)
		if err != nil {
			continue
		}
		info := m.Config.GetModelInfo(provName)
		for _, modelID := range prov.Models() {
			display := modelID
			if modelID == info.ID && info.Name != "" {
				display = info.Name
			}
			models = append(models, ModelItem{
				ID:       modelID,
				Name:     display,
				Provider: provName,
				Context:  info.ContextWindow,
				Selected: modelID == m.Model_ && provName == m.Provider,
			})
		}
	}
	sort.Slice(models, func(i, j int) bool {
		if models[i].Provider == m.Provider && models[j].Provider != m.Provider {
			return true
		}
		if models[i].Provider != m.Provider && models[j].Provider == m.Provider {
			return false
		}
		if models[i].Provider != models[j].Provider {
			return models[i].Provider < models[j].Provider
		}
		return models[i].ID < models[j].ID
	})
	return models
}

func (m *Model) filteredModels() []ModelItem {
	if m.dialogFilter == "" {
		return m.modelList
	}
	f := strings.ToLower(m.dialogFilter)
	var out []ModelItem
	for _, mi := range m.modelList {
		if strings.Contains(strings.ToLower(mi.ID), f) ||
			strings.Contains(strings.ToLower(mi.Name), f) ||
			strings.Contains(strings.ToLower(mi.Provider), f) {
			out = append(out, mi)
		}
	}
	return out
}

func (m Model) updateModelDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	filtered := m.filteredModels()
	switch msg.String() {
	case "esc":
		m.view = m.previousView
		m.dialogFilter = ""
		m.focusTextarea()
		return m, nil
	case "up", "k":
		if m.dialogSelected > 0 {
			m.dialogSelected--
		}
		return m, nil
	case "down", "j":
		if m.dialogSelected < len(filtered)-1 {
			m.dialogSelected++
		}
		return m, nil
	case "enter":
		if m.dialogSelected < len(filtered) {
			sel := filtered[m.dialogSelected]
			m.Provider = sel.Provider
			m.Model_ = sel.ID
			m.view = m.previousView
			m.dialogFilter = ""
			m.focusTextarea()
			m.setStatus(fmt.Sprintf("Model: %s (%s)", sel.Name, sel.Provider))
			return m, nil
		}
		return m, nil
	case "backspace":
		if len(m.dialogFilter) > 0 {
			m.dialogFilter = m.dialogFilter[:len(m.dialogFilter)-1]
			m.dialogSelected = 0
		}
		return m, nil
	default:
		if len(msg.String()) == 1 {
			m.dialogFilter += msg.String()
			m.dialogSelected = 0
		}
		return m, nil
	}
}

// ─── Agent Dialog ───────────────────────────────────────────────────────────────

func agentNames() []string {
	return []string{"coder", "planner", "explorer", "researcher"}
}

func (m Model) cycleAgent(direction int) (tea.Model, tea.Cmd) {
	agents := agentNames()
	cur := 0
	for i, a := range agents {
		if a == m.Agent {
			cur = i
			break
		}
	}
	next := (cur + direction + len(agents)) % len(agents)
	m.Agent = agents[next]
	m.setStatus("Agent: " + m.Agent)
	return m, nil
}

func (m Model) openAgentDialog() (tea.Model, tea.Cmd) {
	m.previousView = m.view
	m.view = ViewAgents
	m.dialogSelected = 0
	agents := agentNames()
	for i, a := range agents {
		if a == m.Agent {
			m.dialogSelected = i
			break
		}
	}
	return m, nil
}

func (m Model) onAgentSelect() (tea.Model, tea.Cmd) {
	agents := agentNames()
	if m.dialogSelected < len(agents) {
		m.Agent = agents[m.dialogSelected]
		m.view = m.previousView
		m.focusTextarea()
		m.setStatus("Agent: " + m.Agent)
	}
	return m, nil
}

// ─── Command Palette ────────────────────────────────────────────────────────────

func (m Model) openCommandPalette() (tea.Model, tea.Cmd) {
	m.previousView = m.view
	m.view = ViewCommandPalette
	m.dialogSelected = 0
	m.dialogFilter = ""
	m.commandList = allCommands()
	m.filteredCmds = m.commandList
	return m, nil
}

func (m *Model) filterCommands() {
	if m.dialogFilter == "" {
		m.filteredCmds = m.commandList
		return
	}
	f := strings.ToLower(m.dialogFilter)
	m.filteredCmds = nil
	for _, c := range m.commandList {
		if strings.Contains(strings.ToLower(c.Title), f) ||
			strings.Contains(strings.ToLower(c.Category), f) ||
			strings.Contains(strings.ToLower(c.Slash), f) {
			m.filteredCmds = append(m.filteredCmds, c)
		}
	}
}

func (m Model) updateCommandPalette(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.view = m.previousView
		m.dialogFilter = ""
		m.focusTextarea()
		return m, nil
	case "up", "k":
		if m.dialogSelected > 0 {
			m.dialogSelected--
		}
		return m, nil
	case "down", "j":
		if m.dialogSelected < len(m.filteredCmds)-1 {
			m.dialogSelected++
		}
		return m, nil
	case "enter":
		if m.dialogSelected < len(m.filteredCmds) {
			return m.executeCommand(m.filteredCmds[m.dialogSelected])
		}
		return m, nil
	case "backspace":
		if len(m.dialogFilter) > 0 {
			m.dialogFilter = m.dialogFilter[:len(m.dialogFilter)-1]
			m.filterCommands()
			m.dialogSelected = 0
		}
		return m, nil
	default:
		if len(msg.String()) == 1 {
			m.dialogFilter += msg.String()
			m.filterCommands()
			m.dialogSelected = 0
		}
		return m, nil
	}
}

func (m Model) executeCommand(cmd Command) (tea.Model, tea.Cmd) {
	m.view = ViewChat
	m.dialogFilter = ""
	m.focusTextarea()
	switch cmd.ID {
	case "model.choose":
		return m.openModelDialog()
	case "provider.connect":
		return m.openProviderDialog()
	case "agent.cycle":
		return m.cycleAgent(1)
	case "session.new":
		return m, m.createSession()
	case "session.list":
		m.view = ViewSessions
		m.sessionList = m.Store.List()
		m.selectedSession = 0
		return m, nil
	case "settings.open":
		return m.openSettings()
	case "help":
		m.view = ViewHelp
		return m, nil
	case "compact":
		if m.sessionID != "" {
			m.Store.Compact(m.sessionID, 10)
			m.setStatus("Session compacted")
		}
		return m, nil
	case "export":
		if m.sessionID != "" {
			data, err := m.Store.Export(m.sessionID)
			if err == nil {
				m.setStatus(fmt.Sprintf("Exported %d bytes", len(data)))
			}
		}
		return m, nil
	case "todo":
		todos := tool.GetSessionTodos(m.sessionID)
		if len(todos) == 0 {
			m.setStatus("No todos")
		} else {
			var lines []string
			for _, t := range todos {
				lines = append(lines, fmt.Sprintf("[%s] %s", t.Status, t.Title))
			}
			m.setStatus(strings.Join(lines, " | "))
		}
		return m, nil
	case "clear":
		m.messages = []session.Message{}
		m.updateViewport()
		return m, nil
	case "quit":
		return m, tea.Quit
	}
	return m, nil
}

// ─── Settings Dialog ────────────────────────────────────────────────────────────

type settingItem struct {
	label  string
	value  string
	action string
}

func (m *Model) settingsItems() []settingItem {
	return []settingItem{
		{"Provider", m.Provider, "provider"},
		{"Model", m.Model_, "model"},
		{"Agent", m.Agent, "agent"},
		{"Theme", m.Config.Theme, "theme"},
		{"Streaming", fmt.Sprintf("%v", m.Config.Streaming), "streaming"},
		{"Max Tokens", fmt.Sprintf("%d", m.Config.MaxTokens), ""},
		{"Auto Title", fmt.Sprintf("%v", m.Config.AutoTitle), ""},
		{"Snapshot", fmt.Sprintf("%v", m.Config.Snapshot), ""},
	}
}

func (m Model) openSettings() (tea.Model, tea.Cmd) {
	m.previousView = m.view
	m.view = ViewSettings
	m.dialogSelected = 0
	return m, nil
}

func (m Model) onSettingsSelect() (tea.Model, tea.Cmd) {
	items := m.settingsItems()
	if m.dialogSelected >= len(items) {
		return m, nil
	}
	item := items[m.dialogSelected]
	switch item.action {
	case "provider":
		return m.openProviderDialog()
	case "model":
		return m.openModelDialog()
	case "agent":
		return m.openAgentDialog()
	case "theme":
		if m.Config.Theme == "dark" {
			m.Config.Theme = "light"
		} else {
			m.Config.Theme = "dark"
		}
		m.setStatus("Theme: " + m.Config.Theme)
	case "streaming":
		m.Config.Streaming = !m.Config.Streaming
		m.setStatus(fmt.Sprintf("Streaming: %v", m.Config.Streaming))
	}
	return m, nil
}

// ─── Session List ───────────────────────────────────────────────────────────────

func (m Model) updateSessionList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.view = ViewChat
		m.focusTextarea()
		return m, nil
	case "up", "k":
		if m.selectedSession > 0 {
			m.selectedSession--
		}
		return m, nil
	case "down", "j":
		if m.selectedSession < len(m.sessionList)-1 {
			m.selectedSession++
		}
		return m, nil
	case "enter":
		if m.selectedSession < len(m.sessionList) {
			m.sessionID = m.sessionList[m.selectedSession].ID
			if sess, err := m.Store.Get(m.sessionID); err == nil {
				m.messages = sess.Messages
			}
			m.view = ViewChat
			m.focusTextarea()
			m.updateViewport()
		}
		return m, nil
	case "d":
		if m.selectedSession < len(m.sessionList) {
			m.Store.Delete(m.sessionList[m.selectedSession].ID)
			m.sessionList = m.Store.List()
			if m.selectedSession >= len(m.sessionList) {
				m.selectedSession = len(m.sessionList) - 1
			}
			if m.selectedSession < 0 {
				m.selectedSession = 0
			}
		}
		return m, nil
	}
	return m, nil
}

// ─── Stream Handling ────────────────────────────────────────────────────────────

func (m Model) handleStreamMsg(msg StreamMsg) (tea.Model, tea.Cmd) {
	switch msg.Event.Type {
	case "text":
		m.streamingText.WriteString(msg.Event.Content)
		m.SetLoadingState(LoadingGenerating, "Generating response", "")
		m.updateViewport()
	case "tool_start":
		m.currentTool = msg.Event.ToolName
		// Track the tool call for display
		m.streamingTools = append(m.streamingTools, streamingToolCall{
			Name:   msg.Event.ToolName,
			Detail: msg.Event.Content,
			Active: true,
		})
		m.SetLoadingState(LoadingToolExecution, "", msg.Event.ToolName)
		m.updateViewport()
	case "tool_end":
		m.currentTool = ""
		// Mark matching tool as completed
		for i := len(m.streamingTools) - 1; i >= 0; i-- {
			if m.streamingTools[i].Name == msg.Event.ToolName && m.streamingTools[i].Active {
				m.streamingTools[i].Active = false
				break
			}
		}
		m.SetLoadingState(LoadingGenerating, "Generating response", "")
		m.updateViewport()
	case "thinking":
		m.streamingThinking.WriteString(msg.Event.Content)
		m.SetLoadingState(LoadingThinking, "Thinking", "")
		m.updateViewport()
	case "retry":
		nextAt := time.Now().Add(30 * time.Second) // default
		if msg.Event.NextAt > 0 {
			nextAt = time.UnixMilli(msg.Event.NextAt)
		}
		m.retryInfo = &retryDisplayInfo{
			Attempt: msg.Event.Attempt,
			Message: msg.Event.Content,
			NextAt:  nextAt,
		}
		retryMsg := fmt.Sprintf("Retrying (attempt %d)", msg.Event.Attempt)
		if msg.Event.Content != "" {
			retryMsg = msg.Event.Content
		}
		m.SetLoadingState(LoadingConnecting, retryMsg, "")
		m.updateViewport()
	case "error":
		m.ClearLoadingState()
		m.setStatus("Error: " + msg.Event.Content)
	case "done":
		m.isStreaming = false
		m.streamingText.Reset()
		m.streamingThinking.Reset()
		m.streamingTools = nil
		m.retryInfo = nil
		m.ClearLoadingState()
		
		// Update token tracking from session
		if m.sessionID != "" {
			if sess, err := m.Store.Get(m.sessionID); err == nil {
				m.messages = sess.Messages
				
				// Update token tracker with last message
				if len(sess.Messages) > 0 {
					lastMsg := sess.Messages[len(sess.Messages)-1]
					if lastMsg.TokensIn > 0 || lastMsg.TokensOut > 0 {
						m.tokenTracker.AddMessage(
							lastMsg.ID,
							lastMsg.TokensIn,
							lastMsg.TokensOut,
							lastMsg.Cost,
						)
					}
				}
			}
		}
		
		m.updateViewport()
		// Auto-scroll to bottom
		m.viewport.GotoBottom()
		return m, nil
	}
	if m.streamCh != nil {
		return m, waitForStream(m.streamCh)
	}
	return m, nil
}

func (m Model) handleDoneMsg() (tea.Model, tea.Cmd) {
	m.isStreaming = false
	m.streamingText.Reset()
	m.streamingThinking.Reset()
	m.streamingTools = nil
	m.retryInfo = nil
	m.ClearLoadingState()
	
	// Update messages and token tracking
	if m.sessionID != "" {
		if sess, err := m.Store.Get(m.sessionID); err == nil {
			m.messages = sess.Messages
			
			// Update token tracker with last message
			if len(sess.Messages) > 0 {
				lastMsg := sess.Messages[len(sess.Messages)-1]
				if lastMsg.TokensIn > 0 || lastMsg.TokensOut > 0 {
					m.tokenTracker.AddMessage(
						lastMsg.ID,
						lastMsg.TokensIn,
						lastMsg.TokensOut,
						lastMsg.Cost,
					)
				}
			}
		}
	}
	
	m.updateViewport()
	// Auto-scroll to bottom
	m.viewport.GotoBottom()
	return m, nil
}

// ─── Slash Commands ─────────────────────────────────────────────────────────────

func (m Model) handleSlashCommand(input string) (tea.Model, tea.Cmd) {
	parts := strings.Fields(input)
	cmd := parts[0]

	switch cmd {
	case "/help":
		m.view = ViewHelp
		return m, nil
	case "/quit", "/exit":
		return m, tea.Quit
	case "/clear":
		m.messages = []session.Message{}
		m.updateViewport()
		return m, nil
	case "/model":
		if len(parts) > 1 {
			m.Model_ = strings.Join(parts[1:], " ")
			m.setStatus("Model: " + m.Model_)
			return m, nil
		}
		return m.openModelDialog()
	case "/provider":
		if len(parts) > 1 {
			m.Provider = parts[1]
			m.Model_ = m.Config.GetDefaultModel(m.Provider)
			m.setStatus("Provider: " + m.Provider)
			return m, nil
		}
		return m.openProviderDialog()
	case "/agent":
		if len(parts) > 1 {
			m.Agent = parts[1]
			m.setStatus("Agent: " + m.Agent)
			return m, nil
		}
		return m.openAgentDialog()
	case "/new":
		return m, m.createSession()
	case "/compact":
		if m.sessionID != "" {
			m.Store.Compact(m.sessionID, 10)
			m.setStatus("Session compacted")
		}
		return m, nil
	case "/export":
		if m.sessionID != "" {
			data, err := m.Store.Export(m.sessionID)
			if err == nil {
				m.setStatus(fmt.Sprintf("Exported %d bytes", len(data)))
			}
		}
		return m, nil
	case "/todo":
		todos := tool.GetSessionTodos(m.sessionID)
		if len(todos) == 0 {
			m.setStatus("No todos")
		} else {
			var lines []string
			for _, t := range todos {
				lines = append(lines, fmt.Sprintf("[%s] %s", t.Status, t.Title))
			}
			m.setStatus(strings.Join(lines, " | "))
		}
		return m, nil
	case "/tokens", "/usage":
		// Show detailed token usage
		if m.tokenTracker == nil {
			m.setStatus("No token tracking available")
			return m, nil
		}
		
		// Display detailed usage in viewport
		detailsView := m.RenderDetailedTokenUsage()
		m.viewport.SetContent(detailsView)
		m.setStatus("Press Esc to return to chat")
		return m, nil
	case "/cost":
		// Show cost information
		if m.tokenTracker == nil || m.tokenTracker.TotalCost == 0 {
			m.setStatus("No cost data available")
			return m, nil
		}
		m.setStatus(fmt.Sprintf("Total cost: $%.4f (In: %s, Out: %s)",
			m.tokenTracker.TotalCost,
			formatTokens(m.tokenTracker.TotalTokensIn),
			formatTokens(m.tokenTracker.TotalTokensOut)))
		return m, nil
	case "/session":
		if len(parts) > 1 {
			switch parts[1] {
			case "list":
				m.view = ViewSessions
				m.sessionList = m.Store.List()
				m.selectedSession = 0
			case "new":
				return m, m.createSession()
			}
		}
		return m, nil
	case "/theme":
		if len(parts) > 1 {
			themeName := parts[1]
			err := m.themeRegistry.SetCurrent(themeName)
			if err != nil {
				m.setStatus("Theme not found: " + themeName)
			} else {
				m.currentTheme = m.themeRegistry.Current()
				// Update components with new theme
				m.syntaxHighlighter = components.NewSyntaxHighlighter(m.currentTheme.SyntaxTheme)
				m.markdownRenderer, _ = components.NewMarkdownRenderer(m.width-4, m.currentTheme.MarkdownTheme)
				m.diffViewer = components.NewDiffViewer(m.width, m.currentTheme.SyntaxTheme)
				m.setStatus("Theme changed to: " + themeName)
			}
		} else {
			// List available themes
			themes := m.themeRegistry.List()
			m.setStatus("Available themes: " + strings.Join(themes, ", "))
		}
		return m, nil
	default:
		m.setStatus("Unknown command: " + cmd)
		return m, nil
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// VIEW
// ═══════════════════════════════════════════════════════════════════════════════

func (m Model) View() string {
	switch m.view {
	case ViewProviders:
		return m.renderOverlay(m.renderProviderDialog())
	case ViewModels:
		return m.renderOverlay(m.renderModelDialog())
	case ViewAgents:
		return m.renderOverlay(m.renderAgentDialog())
	case ViewCommandPalette:
		return m.renderOverlay(m.renderCommandPalette())
	case ViewSettings:
		return m.renderOverlay(m.renderSettingsDialog())
	case ViewSessions:
		return m.renderSessionListView()
	case ViewHelp:
		return m.renderHelpView()
	default:
		return m.renderChat()
	}
}

// ─── Overlay ────────────────────────────────────────────────────────────────────

func (m *Model) renderOverlay(dialog string) string {
	bg := m.renderChat()
	bgLines := strings.Split(bg, "\n")
	dlgLines := strings.Split(dialog, "\n")

	startY := (m.height - len(dlgLines)) / 2
	if startY < 1 {
		startY = 1
	}

	maxW := 0
	for _, l := range dlgLines {
		if w := lipgloss.Width(l); w > maxW {
			maxW = w
		}
	}
	startX := (m.width - maxW) / 2
	if startX < 0 {
		startX = 0
	}

	result := make([]string, len(bgLines))
	for i, line := range bgLines {
		di := i - startY
		if di >= 0 && di < len(dlgLines) {
			result[i] = strings.Repeat(" ", startX) + dlgLines[di]
		} else {
			result[i] = line
		}
	}
	return strings.Join(result, "\n")
}

// ─── Chat ───────────────────────────────────────────────────────────────────────

var (
	// opencode-style left border for messages
	userBorderStyle = lipgloss.NewStyle().
			Border(lipgloss.Border{Left: "┃"}).
			BorderForeground(blue).
			PaddingLeft(2).
			PaddingTop(0).
			PaddingBottom(0)

	assistantBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.Border{Left: "┃"}).
				BorderForeground(purple).
				PaddingLeft(2).
				PaddingTop(0).
				PaddingBottom(0)

	toolBlockBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.Border{Left: "┃"}).
				BorderForeground(surface).
				PaddingLeft(2).
				PaddingTop(0).
				PaddingBottom(0)

	// Tool icons matching opencode
	toolIcons = map[string]struct {
		icon  string
		color lipgloss.Color
	}{
		"glob":     {"✱", yellow},
		"grep":     {"✱", yellow},
		"read":     {"→", blue},
		"ls":       {"→", blue},
		"bash":     {"$", green},
		"write":    {"←", purple},
		"edit":     {"←", purple},
		"patch":    {"←", purple},
		"webfetch": {"%", blue},
		"task":     {"#", purple},
		"todo":     {"☐", green},
	}
)

func (m *Model) renderChat() string {
	if m.width == 0 {
		// Show a minimal loading screen before WindowSizeMsg arrives
		return lipgloss.NewStyle().Bold(true).Foreground(purple).Render(" DCode ") +
			"  " + m.spinner.View() + " Starting...\n"
	}

	var b strings.Builder

	// ── Header: left-aligned info ──
	headerParts := []string{titleStyle.Render(" DCode ")}

	if m.providerInitializing {
		headerParts = append(headerParts, " ", m.spinner.View()+" "+dimStyle.Render("Connecting to "+m.Provider+"..."))
	} else {
		headerParts = append(headerParts,
			" ",
			providerBadge.Render(m.Provider),
			" ",
			modelBadge.Render(shortModel(m.Model_)),
			" ",
			agentBadge.Render(m.Agent),
		)
	}

	// Add message count (always show)
	msgCountStyle := lipgloss.NewStyle().Foreground(m.currentTheme.TextMuted)
	headerParts = append(headerParts, " ", msgCountStyle.Render(fmt.Sprintf("[%d msgs]", len(m.messages))))

	// Add token usage (always show, even if 0)
	if m.tokenTracker != nil {
		headerParts = append(headerParts, " ", m.RenderTokenUsage())
	}

	headerLeft := lipgloss.JoinHorizontal(lipgloss.Center, headerParts...)

	if s := m.getStatus(); s != "" {
		headerLeft += "  " + dimStyle.Render(s)
	}

	b.WriteString(headerLeft + "\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", m.width)) + "\n")

	// ── Messages viewport ──
	if len(m.messages) == 0 && !m.isStreaming {
		// Show welcome message when chat is empty
		welcome := "\n"
		welcome += lipgloss.NewStyle().Foreground(purple).Bold(true).Render("  Welcome to DCode") + "\n\n"
		welcome += dimStyle.Render("  Type a message below and press Enter to start.") + "\n"
		welcome += dimStyle.Render("  Use / for commands, Ctrl+K for model selection.") + "\n"
		welcome += dimStyle.Render("  Press ? for help anytime.") + "\n"
		if m.providerInitError != nil {
			welcome += "\n" + errorStyle.Render("  "+m.providerInitError.Error()) + "\n"
			welcome += dimStyle.Render("  Run `dcode auth login` to set up authentication.") + "\n"
		}
		m.viewport.SetContent(welcome)
	}
	
	// Render viewport with scrollbar
	viewportContent := m.RenderViewportWithScrollbar()
	b.WriteString(viewportContent)
	b.WriteString("\n")

	// ── Loading/Streaming indicator ──
	if m.loadingState.IsActive {
		b.WriteString(m.RenderLoadingState() + "\n")
	} else if m.isStreaming {
		// Fallback to simple streaming indicator
		ind := m.spinner.View() + " "
		if m.currentTool != "" {
			ind += toolCallStyle.Render("⚡ " + m.currentTool)
		} else {
			ind += dimStyle.Render("Generating response...")
		}
		b.WriteString(ind + "\n")
	}

	// ── Input separator + textarea ──
	b.WriteString(dimStyle.Render(strings.Repeat("─", m.width)) + "\n")
	b.WriteString(m.textarea.View())
	b.WriteString("\n")

	// ── Footer: status bar ──
	focusHint := ""
	if m.focusInput {
		focusHint = keybindStyle.Render("[INPUT]") + " "
	} else {
		focusHint = dimStyle.Render("[SCROLL]") + " "
	}
	
	// Add scroll indicator
	scrollIndicator := m.GetScrollIndicator()
	if scrollIndicator != "" {
		focusHint += scrollIndicator + " "
	}

	foot := lipgloss.JoinHorizontal(lipgloss.Center,
		focusHint,
		keybindStyle.Render("Tab")+" focus",
		"  ",
		keybindStyle.Render("Enter")+" send",
		"  ",
		keybindStyle.Render("Ctrl+K")+" model",
		"  ",
		keybindStyle.Render("Ctrl+J")+" agent",
		"  ",
		keybindStyle.Render("Ctrl+P")+" provider",
		"  ",
		keybindStyle.Render("Ctrl+N")+" new",
		"  ",
		keybindStyle.Render("/")+" commands",
	)
	b.WriteString(dimStyle.Render(foot))

	return b.String()
}

// ─── Provider dialog ────────────────────────────────────────────────────────────

func (m *Model) renderProviderDialog() string {
	var b strings.Builder
	w := clampWidth(m.width, 56)

	b.WriteString(dialogTitleStyle.Render("Select Provider") + "\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", w)) + "\n\n")

	// Connected
	hasConn := false
	for i, p := range m.providerList {
		if !p.Connected {
			continue
		}
		if !hasConn {
			b.WriteString(highlightStyle.Render("  Connected") + "\n\n")
			hasConn = true
		}
		m.writeProviderRow(&b, p, i)
	}

	// Available
	hasAvail := false
	for i, p := range m.providerList {
		if p.Connected {
			continue
		}
		if !hasAvail {
			if hasConn {
				b.WriteString("\n")
			}
			b.WriteString(dimStyle.Render("  Available (run `dcode login`)") + "\n\n")
			hasAvail = true
		}
		m.writeProviderRow(&b, p, i)
	}

	b.WriteString("\n" + dimStyle.Render("  Enter: select  Esc: cancel"))
	return dialogBorder.Width(w).Render(b.String())
}

func (m *Model) writeProviderRow(b *strings.Builder, p ProviderInfo, idx int) {
	cur := "  "
	style := unselectedItemStyle
	if idx == m.dialogSelected {
		cur = "▸ "
		style = selectedItemStyle
	}
	badge := dimStyle.Render(" ○")
	if p.Connected {
		badge = successStyle.Render(" ●")
	}
	if p.Name == m.Provider {
		badge += highlightStyle.Render(" ✓")
	}
	b.WriteString(fmt.Sprintf("%s%s%s  %s\n", cur, style.Render(p.DisplayName), badge, dimStyle.Render(p.Description)))
}

// ─── Model dialog ───────────────────────────────────────────────────────────────

func (m *Model) renderModelDialog() string {
	var b strings.Builder
	w := clampWidth(m.width, 66)

	b.WriteString(dialogTitleStyle.Render("Select Model") + "\n")
	if m.dialogFilter != "" {
		b.WriteString(fmt.Sprintf("  🔍 %s\n", m.dialogFilter))
	} else {
		b.WriteString(dimStyle.Render("  Type to search...") + "\n")
	}
	b.WriteString(dimStyle.Render(strings.Repeat("─", w)) + "\n\n")

	filtered := m.filteredModels()
	if len(filtered) == 0 {
		b.WriteString(dimStyle.Render("  No models found.\n"))
		b.WriteString(dimStyle.Render("  Run `dcode login` to connect a provider.\n"))
	}

	curProv := ""
	maxShow := 20
	shown := 0
	for i, mi := range filtered {
		if shown >= maxShow {
			b.WriteString(dimStyle.Render(fmt.Sprintf("\n  ... and %d more", len(filtered)-shown)) + "\n")
			break
		}
		if mi.Provider != curProv {
			curProv = mi.Provider
			b.WriteString(highlightStyle.Render(fmt.Sprintf("  %s", strings.ToUpper(curProv))) + "\n")
		}
		cur := "    "
		style := unselectedItemStyle
		if i == m.dialogSelected {
			cur = "  ▸ "
			style = selectedItemStyle
		}
		sel := ""
		if mi.Selected {
			sel = highlightStyle.Render(" ✓")
		}
		ctx := ""
		if mi.Context > 0 {
			ctx = dimStyle.Render(fmt.Sprintf(" %dk", mi.Context/1000))
		}
		b.WriteString(fmt.Sprintf("%s%s%s%s\n", cur, style.Render(mi.ID), sel, ctx))
		shown++
	}

	b.WriteString("\n" + dimStyle.Render("  Enter: select  Type: filter  Esc: cancel"))
	return dialogBorder.Width(w).Render(b.String())
}

// ─── Agent dialog ───────────────────────────────────────────────────────────────

func (m *Model) renderAgentDialog() string {
	var b strings.Builder
	w := clampWidth(m.width, 50)

	b.WriteString(dialogTitleStyle.Render("Select Agent") + "\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", w)) + "\n\n")

	agents := agentNames()
	builtins := agent.BuiltinAgents()
	for i, name := range agents {
		cur := "  "
		style := unselectedItemStyle
		if i == m.dialogSelected {
			cur = "▸ "
			style = selectedItemStyle
		}
		sel := ""
		if name == m.Agent {
			sel = highlightStyle.Render(" ✓")
		}
		b.WriteString(fmt.Sprintf("%s%s%s\n", cur, style.Render(name), sel))
		if a, ok := builtins[name]; ok {
			b.WriteString(fmt.Sprintf("    %s\n", dimStyle.Render(a.Description)))
		}
	}

	b.WriteString("\n" + dimStyle.Render("  Enter: select  Esc: cancel"))
	b.WriteString("\n" + dimStyle.Render("  Tip: Ctrl+J to cycle agents quickly"))
	return dialogBorder.Width(w).Render(b.String())
}

// ─── Command palette ────────────────────────────────────────────────────────────

func (m *Model) renderCommandPalette() string {
	var b strings.Builder
	w := clampWidth(m.width, 60)

	b.WriteString(dialogTitleStyle.Render("Command Palette") + "\n")
	if m.dialogFilter != "" {
		b.WriteString(fmt.Sprintf("  > %s\n", m.dialogFilter))
	} else {
		b.WriteString(dimStyle.Render("  > Type a command...") + "\n")
	}
	b.WriteString(dimStyle.Render(strings.Repeat("─", w)) + "\n\n")

	curCat := ""
	for i, c := range m.filteredCmds {
		if c.Category != curCat {
			curCat = c.Category
			b.WriteString(dimStyle.Render(fmt.Sprintf("  %s", curCat)) + "\n")
		}
		cur := "    "
		style := unselectedItemStyle
		if i == m.dialogSelected {
			cur = "  ▸ "
			style = selectedItemStyle
		}
		kb := ""
		if c.Keybind != "" {
			kb = keybindStyle.Render(fmt.Sprintf(" [%s]", c.Keybind))
		}
		sl := ""
		if c.Slash != "" {
			sl = dimStyle.Render(fmt.Sprintf(" %s", c.Slash))
		}
		b.WriteString(fmt.Sprintf("%s%s%s%s\n", cur, style.Render(c.Title), kb, sl))
	}

	b.WriteString("\n" + dimStyle.Render("  Enter: run  Type: filter  Esc: cancel"))
	return dialogBorder.Width(w).Render(b.String())
}

// ─── Settings dialog ────────────────────────────────────────────────────────────

func (m *Model) renderSettingsDialog() string {
	var b strings.Builder
	w := clampWidth(m.width, 55)

	b.WriteString(dialogTitleStyle.Render("Settings") + "\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", w)) + "\n\n")

	items := m.settingsItems()
	for i, item := range items {
		cur := "  "
		style := unselectedItemStyle
		if i == m.dialogSelected {
			cur = "▸ "
			style = selectedItemStyle
		}
		arrow := ""
		if item.action != "" {
			arrow = dimStyle.Render(" →")
		}
		b.WriteString(fmt.Sprintf("%s%-20s %s%s\n", cur, style.Render(item.label), highlightStyle.Render(item.value), arrow))
	}

	b.WriteString("\n" + dimStyle.Render("  Enter: change  Esc: back"))
	return dialogBorder.Width(w).Render(b.String())
}

// ─── Session list ───────────────────────────────────────────────────────────────

func (m *Model) renderSessionListView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(" Sessions ") + "\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", m.width)) + "\n\n")

	if len(m.sessionList) == 0 {
		b.WriteString(dimStyle.Render("  No sessions yet. Press Ctrl+N to create one.") + "\n")
	}
	for i, sess := range m.sessionList {
		cur := "  "
		style := dimStyle
		if i == m.selectedSession {
			cur = "▸ "
			style = highlightStyle
		}
		title := sess.Title
		if len(title) > 50 {
			title = title[:47] + "..."
		}
		b.WriteString(fmt.Sprintf("%s%s  %s  %d msgs  %s\n",
			cur,
			style.Render(title),
			dimStyle.Render(sess.Agent),
			len(sess.Messages),
			dimStyle.Render(timeAgo(sess.UpdatedAt)),
		))
	}

	b.WriteString("\n" + dimStyle.Render("Enter: select | D: delete | Esc: back"))
	return b.String()
}

// ─── Help ───────────────────────────────────────────────────────────────────────

func (m *Model) renderHelpView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(" DCode Help ") + "\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", m.width)) + "\n\n")

	b.WriteString(highlightStyle.Render("  Keyboard Shortcuts") + "\n\n")
	shortcuts := []struct{ key, desc string }{
		{"?", "Show this help (works anywhere)"},
		{"Enter", "Send message"},
		{"Alt+Enter", "Insert newline"},
		{"Tab", "Toggle focus (input/viewport)"},
		{"Ctrl+Y", "Copy last assistant message"},
		{"Ctrl+K", "Select model"},
		{"Ctrl+J", "Cycle agent forward"},
		{"Ctrl+P", "Select provider"},
		{"Ctrl+N", "New session"},
		{"Ctrl+L", "Toggle sessions"},
		{"Ctrl+Shift+L", "Clear screen"},
		{"Ctrl+Shift+P", "Command palette"},
		{"Ctrl+S", "Settings"},
		{"Esc", "Cancel / Back"},
		{"Ctrl+C", "Quit"},
	}
	for _, s := range shortcuts {
		b.WriteString(fmt.Sprintf("  %s  %s\n",
			keybindStyle.Render(fmt.Sprintf("%-15s", s.key)),
			descStyle.Render(s.desc),
		))
	}

	b.WriteString("\n" + highlightStyle.Render("  Slash Commands") + "\n\n")
	commands := []struct{ cmd, desc string }{
		{"/model [name]", "Select/switch model (dialog if no arg)"},
		{"/provider [name]", "Select/switch provider (dialog if no arg)"},
		{"/agent [name]", "Select/switch agent (dialog if no arg)"},
		{"/new", "Create new session"},
		{"/session list", "List all sessions"},
		{"/compact", "Compact session (save context)"},
		{"/export", "Export session as JSON"},
		{"/todo", "Show current todos"},
		{"/tokens or /usage", "Show detailed token usage"},
		{"/cost", "Show cost information"},
		{"/clear", "Clear screen"},
		{"/help", "Show this help"},
		{"/quit", "Exit DCode"},
	}
	for _, c := range commands {
		b.WriteString(fmt.Sprintf("  %s  %s\n",
			keybindStyle.Render(fmt.Sprintf("%-20s", c.cmd)),
			descStyle.Render(c.desc),
		))
	}

	b.WriteString("\n" + highlightStyle.Render("  Agents") + "\n\n")
	builtins := agent.BuiltinAgents()
	for _, name := range agentNames() {
		a := builtins[name]
		if a == nil {
			continue
		}
		marker := "  "
		if name == m.Agent {
			marker = "▸ "
		}
		b.WriteString(fmt.Sprintf("%s%s  %s\n", marker, keybindStyle.Render(fmt.Sprintf("%-12s", name)), descStyle.Render(a.Description)))
	}

	b.WriteString("\n" + highlightStyle.Render("  Text Copying") + "\n\n")
	b.WriteString(dimStyle.Render("  Due to terminal limitations with mouse capture:") + "\n\n")
	copyFeatures := []struct{ feature, desc string }{
		{"Ctrl+Y", "Copy last assistant message to clipboard"},
		{"Shift+Mouse", "Select text (bypasses app, terminal-dependent)"},
		{"Mouse Wheel", "Scroll messages up/down"},
		{"Click", "Focus input or viewport"},
		{"Scrollbar", "Visual indicator on right side"},
	}
	for _, f := range copyFeatures {
		b.WriteString(fmt.Sprintf("  %s  %s\n",
			keybindStyle.Render(fmt.Sprintf("%-15s", f.feature)),
			descStyle.Render(f.desc),
		))
	}
	b.WriteString("\n" + dimStyle.Render("  Note: Shift+Mouse selection works in most terminals (iTerm2,") + "\n")
	b.WriteString(dimStyle.Render("  Windows Terminal, Alacritty). Use Ctrl+Y for quick copy.") + "\n")

	b.WriteString("\n" + dimStyle.Render("Press Esc or Q to go back"))
	return b.String()
}

// ─── Viewport content ───────────────────────────────────────────────────────────

// getToolIcon returns the icon and color for a tool name
func getToolIcon(toolName string) (string, lipgloss.Color) {
	if info, ok := toolIcons[toolName]; ok {
		return info.icon, info.color
	}
	return "⚙", overlay
}

// formatToolInput returns a compact summary of tool input params
func formatToolInput(toolName string, input map[string]interface{}) string {
	switch toolName {
	case "glob":
		if p, ok := input["pattern"]; ok {
			s := fmt.Sprintf("\"%v\"", p)
			if d, ok := input["directory"]; ok {
				s += fmt.Sprintf(" in %v", d)
			}
			return s
		}
	case "grep":
		if p, ok := input["pattern"]; ok {
			s := fmt.Sprintf("\"%v\"", p)
			if d, ok := input["path"]; ok {
				s += fmt.Sprintf(" in %v", d)
			}
			return s
		}
	case "read":
		if f, ok := input["file_path"]; ok {
			return fmt.Sprintf("%v", f)
		}
	case "ls":
		if d, ok := input["path"]; ok {
			return fmt.Sprintf("%v", d)
		}
		if d, ok := input["directory"]; ok {
			return fmt.Sprintf("%v", d)
		}
	case "bash":
		if c, ok := input["command"]; ok {
			cmd := fmt.Sprintf("%v", c)
			if len(cmd) > 60 {
				cmd = cmd[:57] + "..."
			}
			return cmd
		}
	case "write":
		if f, ok := input["file_path"]; ok {
			return fmt.Sprintf("%v", f)
		}
	case "edit", "patch":
		if f, ok := input["file_path"]; ok {
			return fmt.Sprintf("%v", f)
		}
	case "webfetch":
		if u, ok := input["url"]; ok {
			return fmt.Sprintf("%v", u)
		}
	case "task":
		if d, ok := input["description"]; ok {
			desc := fmt.Sprintf("%v", d)
			if len(desc) > 50 {
				desc = desc[:47] + "..."
			}
			return desc
		}
	}
	return ""
}

// truncateOutput truncates long tool output with a summary line
func truncateOutput(content string, maxLines int) string {
	lines := strings.Split(content, "\n")
	if len(lines) <= maxLines {
		return content
	}
	result := strings.Join(lines[:maxLines], "\n")
	result += "\n" + dimStyle.Render(fmt.Sprintf("... %d more lines (Click to expand)", len(lines)-maxLines))
	return result
}

func (m *Model) updateViewport() {
	var content strings.Builder
	contentWidth := m.width - 6 // account for border + padding
	if contentWidth < 20 {
		contentWidth = 20
	}

	for idx, msg := range m.messages {
		switch msg.Role {
		case "user":
			m.renderUserMessage(&content, msg, idx, contentWidth)
		case "assistant":
			m.renderAssistantMessage(&content, msg, contentWidth)
		}
	}

	// Streaming content
	if m.isStreaming {
		var streamContent strings.Builder

		// 1. Show thinking block (dimmed, italic)
		if m.streamingThinking.Len() > 0 {
			thinkText := m.streamingThinking.String()
			thinkStyle := lipgloss.NewStyle().Foreground(m.currentTheme.TextDim).Italic(true)
			streamContent.WriteString(thinkStyle.Render("💭 Thinking...") + "\n")
			// Truncate to last ~500 chars to avoid viewport bloat
			if len(thinkText) > 500 {
				thinkText = "..." + thinkText[len(thinkText)-497:]
			}
			streamContent.WriteString(thinkStyle.Render(thinkText) + "\n\n")
		}

		// 2. Show retry info
		if m.retryInfo != nil {
			retryStyle := lipgloss.NewStyle().Foreground(m.currentTheme.Warning)
			remaining := time.Until(m.retryInfo.NextAt)
			if remaining < 0 {
				remaining = 0
			}
			retryMsg := m.retryInfo.Message
			if retryMsg == "" {
				retryMsg = "Retrying"
			}
			streamContent.WriteString(retryStyle.Render(
				fmt.Sprintf("⟳ %s (attempt %d, retrying in %s)",
					retryMsg, m.retryInfo.Attempt, formatDuration(remaining)),
			) + "\n\n")
		}

		// 3. Show text content
		if m.streamingText.Len() > 0 {
			streamContent.WriteString(m.streamingText.String() + dimStyle.Render("▊"))
		}

		// 4. Show tool calls
		for _, tc := range m.streamingTools {
			icon, clr := getToolIcon(tc.Name)
			iconStyled := lipgloss.NewStyle().Foreground(clr).Render(icon)
			status := successStyle.Render("✓")
			if tc.Active {
				status = toolCallStyle.Render("⟳")
			}
			line := fmt.Sprintf("   %s %s %s", iconStyled, highlightStyle.Render(tc.Name), status)
			if tc.Detail != "" {
				line += " " + dimStyle.Render(tc.Detail)
			}
			streamContent.WriteString("\n" + line)
		}

		if streamContent.Len() > 0 {
			bordered := assistantBorderStyle.Width(contentWidth).Render(streamContent.String())
			content.WriteString(bordered + "\n")
		}
	}

	m.viewport.SetContent(content.String())
	m.viewport.GotoBottom()
}

func (m *Model) renderUserMessage(b *strings.Builder, msg session.Message, idx int, width int) {
	if msg.Content == "" && len(msg.Parts) == 0 {
		return
	}

	var inner strings.Builder

	if msg.Content != "" {
		inner.WriteString(userMsgStyle.Render(msg.Content))
	}

	// Tool results (from previous assistant tool calls)
	for _, part := range msg.Parts {
		if part.Type == "tool_result" {
			status := successStyle.Render("✓")
			if part.IsError {
				status = errorStyle.Render("✗")
			}
			inner.WriteString("\n" + status + " " + dimStyle.Render("result"))
			if part.Content != "" {
				truncated := truncateOutput(part.Content, 8)
				inner.WriteString("\n" + dimStyle.Render(truncated))
			}

			// Render side-by-side diff if available
			if part.Metadata != nil && m.diffViewer != nil {
				diffs := extractDiffDataFromMetadata(part.Metadata)
				for _, dd := range diffs {
					diffView := m.diffViewer.RenderEditDiff(dd.OldContent, dd.NewContent, dd.FilePath, 30)
					inner.WriteString("\n" + diffView)
				}
			}
		}
	}

	marginTop := ""
	if idx > 0 {
		marginTop = "\n"
	}
	bordered := userBorderStyle.Width(width).Render(inner.String())
	b.WriteString(marginTop + bordered + "\n")
}

func (m *Model) renderAssistantMessage(b *strings.Builder, msg session.Message, width int) {
	// Render text content with markdown and syntax highlighting
	if msg.Content != "" {
		rendered := msg.Content

		// Enhanced markdown rendering with syntax highlighting
		if m.markdownRenderer != nil && m.syntaxHighlighter != nil {
			// First, highlight code blocks using syntax highlighter
			codeBlocks := components.FindCodeBlocks(msg.Content)
			if len(codeBlocks) > 0 {
				// Replace code blocks with highlighted versions
				processedContent := msg.Content
				for i := len(codeBlocks) - 1; i >= 0; i-- {
					block := codeBlocks[i]
					highlighted := m.syntaxHighlighter.Highlight(block.Code, block.Language)

					// Create styled code block
					codeStyle := lipgloss.NewStyle().
						Border(lipgloss.RoundedBorder()).
						BorderForeground(overlay).
						Padding(1, 2).
						Foreground(txtClr)

					styledCode := codeStyle.Render(highlighted)

					// Replace original code block
					original := "```" + block.Language + "\n" + block.Code + "\n```"
					processedContent = strings.Replace(processedContent, original, "\n"+styledCode+"\n", 1)
				}
				rendered = processedContent
			} else if strings.Contains(msg.Content, "#") || strings.Contains(msg.Content, "*") ||
				strings.Contains(msg.Content, "-") || strings.Contains(msg.Content, ">") {
				// Render as markdown if it has markdown syntax
				renderedMd, err := m.markdownRenderer.Render(msg.Content)
				if err == nil {
					rendered = renderedMd
				}
			}
		}

		bordered := assistantBorderStyle.Width(width).Render(
			assistantMsgStyle.Render(rendered),
		)
		b.WriteString("\n" + bordered + "\n")
	}

	// Render tool calls (opencode-style inline tools)
	for _, part := range msg.Parts {
		switch part.Type {
		case "tool_use":
			icon, clr := getToolIcon(part.ToolName)
			iconStyled := lipgloss.NewStyle().Foreground(clr).Render(icon)
			summary := formatToolInput(part.ToolName, part.ToolInput)

			toolLine := fmt.Sprintf("   %s %s", iconStyled, highlightStyle.Render(part.ToolName))
			if summary != "" {
				toolLine += " " + dimStyle.Render(summary)
			}

			// Status indicator
			switch part.Status {
			case "running", "pending":
				toolLine += " " + toolCallStyle.Render("⟳")
			case "error":
				toolLine += " " + errorStyle.Render("✗")
			case "completed":
				toolLine += " " + successStyle.Render("✓")
			}

			b.WriteString(toolLine + "\n")

		case "reasoning":
			if part.Content != "" {
				thinkContent := dimStyle.Render("Thinking: " + part.Content)
				bordered := toolBlockBorderStyle.Width(width).Render(thinkContent)
				b.WriteString(bordered + "\n")
			}

		case "error":
			errContent := errorStyle.Render("Error: " + part.Content)
			bordered := assistantBorderStyle.Width(width).Render(errContent)
			b.WriteString(bordered + "\n")
		}
	}

	// Completion footer (opencode style: ▣ Agent · model · duration)
	if !m.isStreaming && msg.Content != "" {
		footer := lipgloss.NewStyle().Foreground(purple).Render("▣") + " " +
			lipgloss.NewStyle().Foreground(txtClr).Render(strings.Title(m.Agent))
		footer += dimStyle.Render(" · " + shortModel(m.Model_))
		if msg.TokensIn > 0 || msg.TokensOut > 0 {
			footer += dimStyle.Render(fmt.Sprintf(" · %d→%d tokens", msg.TokensIn, msg.TokensOut))
		}
		b.WriteString("   " + footer + "\n")
	}
}

// ─── Session & Engine ───────────────────────────────────────────────────────────

func (m *Model) createSession() tea.Cmd {
	return func() tea.Msg {
		sess, err := m.Store.Create(m.Agent, m.Model_, m.Provider)
		if err != nil {
			return ErrorMsg{Error: err}
		}
		return SessionCreatedMsg{Session: sess}
	}
}

func (m *Model) sendMessage(input string) tea.Cmd {
	// Initialize engine if not already done
	if m.Engine == nil {
		if m.providerInitializing {
			m.setStatus("Provider initializing, please wait...")
			return nil
		}
		if m.providerInitError != nil {
			// Keep the existing error message visible
			return nil
		}
		// Try to initialize the engine now
		m.providerInitializing = true
		m.providerInitError = nil
		apiKey, keyErr := config.GetAPIKeyWithFallback(m.Provider, m.Config)
		if keyErr != nil {
			err := fmt.Errorf("no API key for %s: %w. Run: dcode login", m.Provider, keyErr)
			m.providerInitError = err
			m.providerInitializing = false
			m.setStatus(err.Error())
			return nil
		}
		prov, err := provider.CreateProvider(m.Provider, apiKey)
		if err != nil {
			err = fmt.Errorf("failed to create provider %s: %w", m.Provider, err)
			m.providerInitError = err
			m.providerInitializing = false
			m.setStatus(err.Error())
			return nil
		}
		ag := agent.GetAgent(m.Agent, m.Config)
		registry := tool.GetRegistry()
		m.Engine = session.NewPromptEngine(m.Store, prov, m.Config, ag, registry)
		m.providerInitializing = false
		m.providerInitError = nil
	}

	m.isStreaming = true
	m.streamingText.Reset()
	m.streamingThinking.Reset()
	m.streamingTools = nil
	m.retryInfo = nil

	m.messages = append(m.messages, session.Message{
		Role:    "user",
		Content: input,
	})
	m.updateViewport()

	if m.sessionID == "" {
		sess, err := m.Store.Create(m.Agent, m.Model_, m.Provider)
		if err != nil {
			return func() tea.Msg { return ErrorMsg{Error: err} }
		}
		m.sessionID = sess.ID
	}

	ch := make(chan tea.Msg, 256)
	m.streamCh = ch

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	m.Engine.OnStream(func(event session.StreamEvent) {
		ch <- StreamMsg{Event: event}
	})

	go func() {
		defer close(ch)
		err := m.Engine.Run(ctx, m.sessionID, input)
		if err != nil && ctx.Err() == nil {
			ch <- ErrorMsg{Error: err}
		}
	}()

	return waitForStream(ch)
}

func waitForStream(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return DoneMsg{}
		}
		return msg
	}
}

func (m *Model) reinitEngine() tea.Cmd {
	return func() tea.Msg {
		apiKey, keyErr := config.GetAPIKeyWithFallback(m.Provider, m.Config)
		if keyErr != nil {
			return ErrorMsg{Error: fmt.Errorf("no API key for %s: %w", m.Provider, keyErr)}
		}
		prov, err := provider.CreateProvider(m.Provider, apiKey)
		if err != nil {
			return ErrorMsg{Error: fmt.Errorf("failed to create provider %s: %w", m.Provider, err)}
		}
		ag := agent.GetAgent(m.Agent, m.Config)
		registry := tool.GetRegistry()
		m.Engine = session.NewPromptEngine(m.Store, prov, m.Config, ag, registry)
		return ProviderChangedMsg{Provider: m.Provider, Model: m.Model_}
	}
}

// ─── Helpers ────────────────────────────────────────────────────────────────────

func (m *Model) setStatus(msg string) {
	m.statusMsg = msg
	m.statusExpiry = time.Now().Add(5 * time.Second)
}

func (m *Model) getStatus() string {
	if m.statusMsg != "" && time.Now().Before(m.statusExpiry) {
		return m.statusMsg
	}
	if m.statusMsg != "" {
		m.statusMsg = ""
	}
	return ""
}

func timeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func shortModel(model string) string {
	if len(model) > 30 {
		return model[:27] + "..."
	}
	return model
}

// extractDiffDataFromMetadata extracts DiffData from part metadata,
// handling both typed (in-memory) and JSON-deserialized forms.
func extractDiffDataFromMetadata(metadata map[string]interface{}) []*tool.DiffData {
	var result []*tool.DiffData

	// Try single diff_data
	if raw, ok := metadata["diff_data"]; ok {
		if dd := convertToDiffData(raw); dd != nil {
			result = append(result, dd)
		}
	}

	// Try diff_data_list
	if raw, ok := metadata["diff_data_list"]; ok {
		switch v := raw.(type) {
		case []*tool.DiffData:
			result = append(result, v...)
		case []interface{}:
			for _, item := range v {
				if dd := convertToDiffData(item); dd != nil {
					result = append(result, dd)
				}
			}
		}
	}

	return result
}

// convertToDiffData converts a value to *tool.DiffData, handling both
// typed structs and map[string]interface{} from JSON deserialization.
func convertToDiffData(raw interface{}) *tool.DiffData {
	switch v := raw.(type) {
	case *tool.DiffData:
		return v
	case tool.DiffData:
		return &v
	case map[string]interface{}:
		dd := &tool.DiffData{}
		if s, ok := v["old_content"].(string); ok {
			dd.OldContent = s
		}
		if s, ok := v["new_content"].(string); ok {
			dd.NewContent = s
		}
		if s, ok := v["file_path"].(string); ok {
			dd.FilePath = s
		}
		if s, ok := v["language"].(string); ok {
			dd.Language = s
		}
		if b, ok := v["is_fragment"].(bool); ok {
			dd.IsFragment = b
		}
		if dd.OldContent != "" || dd.NewContent != "" {
			return dd
		}
	default:
		// Try JSON round-trip as last resort
		data, err := json.Marshal(raw)
		if err == nil {
			var dd tool.DiffData
			if json.Unmarshal(data, &dd) == nil && (dd.OldContent != "" || dd.NewContent != "") {
				return &dd
			}
		}
	}
	return nil
}

func clampWidth(screenW, maxW int) int {
	if screenW > maxW+10 {
		return maxW
	}
	if screenW > 30 {
		return screenW - 10
	}
	return 20
}
