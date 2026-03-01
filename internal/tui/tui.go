package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Dhanuzh/dcode/internal/agent"
	"github.com/Dhanuzh/dcode/internal/config"
	"github.com/Dhanuzh/dcode/internal/permission"
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
	ViewOAuthCode      View = "oauth_code"
	ViewTheme          View = "theme"
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
	Engine   *session.PromptEngine // new engine, ready to use
}

// CopilotStatusMsg carries the result of a background Copilot status fetch
type CopilotStatusMsg struct {
	Info *provider.CopilotStatusInfo
}

// ModelsRefreshedMsg is sent when the background model registry refresh finishes
type ModelsRefreshedMsg struct{}

// ─── Streaming display types ────────────────────────────────────────────────────

// streamingToolCall tracks a tool call during streaming
type streamingToolCall struct {
	Name   string
	Detail string
	Active bool
	Diffs  []*tool.DiffData // realtime diffs captured on tool_end
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
	EnvVar      []string // env vars for API key (for help text)
	Priority    int      // lower = shown first; 0=popular
}

// ModelItem holds display data for the model selection dialog
type ModelItem struct {
	ID           string
	Name         string
	Provider     string
	ProviderName string // friendly display name of the provider
	Context      int
	Selected     bool
	IsFree       bool    // cost.input == 0
	CostInput    float64 // $/M tokens input
	CostOutput   float64 // $/M tokens output
	HasReasoning bool
	HasVision    bool
	IsRecent     bool
}

// Command defines an entry in the command palette
type Command struct {
	ID       string
	Title    string
	Category string
	Keybind  string
	Slash    string
}

// allCommands returns the full list of command palette entries.
// The Copilot login entry title is adjusted to reflect current auth status.
func allCommands() []Command {
	// Check Copilot login status (fast: only reads the cached token file)
	copilotTitle := "GitHub Copilot — OAuth Login"
	if _, err := provider.NewCopilotProvider(); err == nil {
		copilotTitle = "GitHub Copilot — Logged in (select model or re-login)"
	}

	return []Command{
		{ID: "model.choose", Title: "Select Model", Category: "Model", Keybind: "Ctrl+K", Slash: "/model"},
		{ID: "provider.connect", Title: "Connect Provider", Category: "Provider", Keybind: "Ctrl+Shift+P", Slash: "/provider"},
		{ID: "copilot.login", Title: copilotTitle, Category: "Provider", Slash: "/copilot-login"},
		{ID: "agent.cycle", Title: "Cycle Agent (Tab/Shift+Tab)", Category: "Agent", Keybind: "Tab", Slash: "/agent"},
		{ID: "session.new", Title: "New Session", Category: "Session", Keybind: "Ctrl+N", Slash: "/new"},
		{ID: "session.list", Title: "List Sessions", Category: "Session", Keybind: "Ctrl+L"},
		{ID: "theme.change", Title: "Change Theme", Category: "General", Slash: "/theme"},
		{ID: "settings.open", Title: "Settings", Category: "General", Keybind: "Ctrl+S"},
		{ID: "help", Title: "Help", Category: "General", Slash: "/help"},
		{ID: "compact", Title: "Compact Session", Category: "Session", Slash: "/compact"},
		{ID: "export", Title: "Export Session", Category: "Session", Slash: "/export"},
		{ID: "todo", Title: "Show Todos", Category: "General", Slash: "/todo"},
		{ID: "undo", Title: "Undo Last Change", Category: "Session", Keybind: "Ctrl+Z", Slash: "/undo"},
		{ID: "redo", Title: "Redo Last Change", Category: "Session", Keybind: "Ctrl+Shift+Z", Slash: "/redo"},
		{ID: "copy", Title: "Copy Code Block", Category: "General", Keybind: "Ctrl+Y", Slash: "/copy"},
		{ID: "clear", Title: "Clear Screen", Category: "General", Slash: "/clear"},
		{ID: "quit", Title: "Quit", Category: "General", Keybind: "Ctrl+C", Slash: "/quit"},
		{ID: "login", Title: "Login / Auth (Claude Pro/Max OAuth)", Category: "General", Slash: "/login"},
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
	view              View
	previousView      View
	width             int
	height            int
	sessionID         string
	messages          []session.Message
	streamingText     *strings.Builder
	streamingThinking *strings.Builder
	streamingTools    []streamingToolCall
	retryInfo         *retryDisplayInfo
	isStreaming       bool
	currentTool       string
	statusMsg         string
	statusExpiry      time.Time
	focusInput        bool // true = textarea focused, false = viewport focused

	// Provider initialization state
	providerInitializing bool
	providerInitError    error

	// Dialog state
	dialogSelected  int
	dialogFilter    string
	sessionList     []*session.Session
	selectedSession int
	renameBuffer    string // inline rename input for session list
	renamingSession bool   // true while typing a new session name

	// Provider/model state
	providerList       []ProviderInfo
	modelList          []ModelItem
	commandList        []Command
	filteredCmds       []Command
	modelRegistry      *provider.ModelRegistry // rich model metadata
	recentModels       []string                // "provider/modelID" MRU list (max 5)
	modelDisplayOrder  []int                   // maps display position → index into filtered models list
	providerActionMode bool                    // true when showing logout/select-model sub-menu
	providerActionIdx  int                     // 0=Select Model, 1=Logout

	// OAuth dialog state
	oauthDialog   *components.Dialog
	oauthVerifier string // PKCE verifier kept while dialog is open

	// Dependencies
	Store    *session.Store
	Engine   *session.PromptEngine
	Config   *config.Config
	Agent    string
	Model_   string
	Provider string
	Todos    []tool.TodoItem

	// Sidebar / footer status counts (updated after each stream event)
	lspCount           int
	mcpCount           int
	mcpErrors          bool
	pendingPermissions int

	// Mouse mode (can be toggled with Ctrl+M so users can copy terminal text)
	mouseEnabled bool

	// Input history (up/down arrow cycling)
	promptHistory *PromptHistory

	// Multi-step undo/redo stack (file snapshots)
	undoStack *UndoRedoStack

	// Code blocks from the last rendered assistant message (for copy)
	lastCodeBlocks []string
	// copyPending: when true, next digit key selects which code block to copy
	copyPending bool

	// Autocomplete popup state
	autocomplete AutocompleteState

	// Toast notifications
	toasts []Toast

	// Permission prompt overlay
	permission PermissionPromptState

	// Question prompt overlay
	questionState QuestionState

	// Theme picker dialog
	themeDialog *themeDialogState

	// Streaming
	streamCh chan tea.Msg
	cancel   context.CancelFunc

	// Token tracking and loading states
	tokenTracker *TokenUsageTracker
	loadingState LoadingState

	// Vim keybinding state
	vimState VimState

	// Last welcome screen content — used to avoid redundant SetContent calls
	lastWelcomeContent string

	// Images staged for the next message send
	pendingImages []session.ImageAttachment
}

// New creates a new TUI model (opencode-style)
func New(store *session.Store, engine *session.PromptEngine, cfg *config.Config, agentName, modelName, prov string) Model {
	ta := textarea.New()
	ta.Placeholder = "Message DCode... (Enter to send, / for commands)"
	ta.Focus()
	ta.CharLimit = 50000
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	// Blinking light-green block cursor on a dark-grey line background.
	//
	// The textarea overrides Cursor.TextStyle with computedCursorLine() on
	// every render (bubbles source line 1095), so FocusedStyle.CursorLine
	// controls what colour the character *under* the cursor gets. We set it
	// to the dark-grey line colour so the rest of the line stays dark while
	// the cursor block itself (Cursor.Style) shows as light green.
	const cursorGreen = "#A6E3A1" // cursor block colour
	const cursorDark = "#2A2A35"  // dark grey line background
	const cursorText = "#1E1E2E"  // text on top of green cursor block

	ta.Cursor.SetMode(cursor.CursorBlink)
	// The block itself: bright green background, dark text inside
	ta.Cursor.Style = lipgloss.NewStyle().
		Background(lipgloss.Color(cursorGreen)).
		Foreground(lipgloss.Color(cursorText))
	// CursorLine: dark grey bg — this is what the textarea applies to the
	// whole active line AND to Cursor.TextStyle; keeping it dark means only
	// the 1-char cursor block (Cursor.Style) stands out as green.
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle().
		Background(lipgloss.Color(cursorDark))
	// Cursor.TextStyle is overridden by computedCursorLine() at render time,
	// but set it explicitly here too for completeness.
	ta.Cursor.TextStyle = lipgloss.NewStyle().
		Background(lipgloss.Color(cursorGreen)).
		Foreground(lipgloss.Color(cursorText))

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

	// Initialize model registry (loads builtin + cache)
	modelReg := provider.NewModelRegistry()

	return Model{
		viewport:          vp,
		textarea:          ta,
		spinner:           sp,
		view:              ViewChat,
		focusInput:        true,
		streamingText:     &strings.Builder{},
		streamingThinking: &strings.Builder{},
		streamingTools:    nil,
		retryInfo:         nil,
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
		promptHistory:     NewPromptHistory(),
		modelRegistry:     modelReg,
		mouseEnabled:      false,
		undoStack:         NewUndoRedoStack(),
		lastCodeBlocks:    []string{},
	}
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		textarea.Blink,
		m.spinner.Tick,
		// Mouse support is provided by tea.WithMouseCellMotion() in NewProgram.
		// Do NOT add tea.EnableMouseAllMotion here — it sends extra terminal
		// queries whose responses can land in the input buffer as garbage text.
		refreshModelsCmd(m.modelRegistry),
	}
	// Initialize provider asynchronously so the TUI renders immediately
	if m.Engine == nil {
		cmds = append(cmds, m.initEngineAsync())
	}
	return tea.Batch(cmds...)
}

// refreshModelsCmd starts a background model registry refresh and returns a
// ModelsRefreshedMsg when it completes so the TUI can update its model list.
func refreshModelsCmd(registry *provider.ModelRegistry) tea.Cmd {
	return func() tea.Msg {
		_ = registry.Refresh()
		return ModelsRefreshedMsg{}
	}
}

// ProviderInitStartMsg signals that async provider initialization has started
type ProviderInitStartMsg struct{}

// fetchCopilotStatusCmd returns a Cmd that fetches Copilot auth/subscription status in the background.
func fetchCopilotStatusCmd() tea.Cmd {
	return func() tea.Msg {
		prov, err := provider.NewCopilotProvider()
		if err != nil {
			return CopilotStatusMsg{Info: &provider.CopilotStatusInfo{Error: err.Error()}}
		}
		return CopilotStatusMsg{Info: prov.GetCopilotStatus()}
	}
}

// initEngineAsync starts provider init and first sends a message to set the flag on the real model
func (m *Model) initEngineAsync() tea.Cmd {
	return tea.Batch(
		func() tea.Msg { return ProviderInitStartMsg{} },
		m.reinitEngine(),
	)
}

// calcViewportHeight returns the correct viewport height given current state.
// Fixed overhead (lines that always appear in renderChat):
//
//	header:  2  (title row + separator row)
//	newline: 1  (after viewport)
//	inpsep:  1  (input separator "────")
//	textarea:3  (SetHeight(3))
//	newline: 1  (after textarea)
//	footer:  3  (sep + cwd/status + hints)
//	total:  11
//
// Dynamic: streaming/loading indicator adds 1 extra line.
func (m *Model) calcViewportHeight() int {
	h := m.height - 11
	if m.isStreaming || m.loadingState.IsActive {
		h--
	}
	if h < 3 {
		h = 3
	}
	return h
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
		// Chat always fills full terminal width (no sidebar).
		chatW := msg.Width
		m.viewport.Width = chatW - 2 // reserve 2 cols for scrollbar
		m.viewport.Height = m.calcViewportHeight()
		m.textarea.SetWidth(chatW - 4)

		// Update component widths — match content width inside the bordered viewport
		if m.markdownRenderer != nil {
			contentW := m.viewport.Width - 4
			if contentW < 20 {
				contentW = 20
			}
			m.markdownRenderer.SetWidth(contentW)
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
		// Global help shortcut – suppress when typing in vim INSERT mode
		if msg.String() == "?" {
			inInsertTyping := m.view == ViewChat && m.focusInput && m.vimState.Mode == VimModeInsert
			if !inInsertTyping {
				m.previousView = m.view
				m.view = ViewHelp
				m.blurTextarea()
				return m, nil
			}
		}
		// Dispatch to current view
		switch m.view {
		case ViewChat:
			return m.updateChat(msg)
		case ViewProviders:
			return m.updateProviderDialog(msg)
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
		case ViewOAuthCode:
			return m.updateOAuthCodeDialog(msg)
		case ViewTheme:
			return m.updateThemeDialog(msg)
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
		m.lastWelcomeContent = ""
		// Reset undo/redo history for the new session
		if m.undoStack != nil {
			m.undoStack.Reset()
		}
		m.setStatus("Session created")
		return m, nil

	case PermissionRequestMsg:
		return m.handlePermissionMsg(msg)

	case QuestionRequestMsg:
		return m.handleQuestionMsg(msg)

	case ErrorMsg:
		m.isStreaming = false
		m.viewport.Height = m.calcViewportHeight()
		m.streamingThinking.Reset()
		m.streamingTools = nil
		m.retryInfo = nil
		// If this is a provider init error, store it and make it permanent
		if m.providerInitializing {
			m.providerInitializing = false
			m.providerInitError = msg.Error
			cmds = append(cmds, m.showToast("Error: "+msg.Error.Error(), ToastError, 8*time.Second))
		} else {
			cmds = append(cmds, m.showToast("Error: "+msg.Error.Error(), ToastError, 6*time.Second))
		}
		return m, tea.Batch(cmds...)

	case ProviderInitStartMsg:
		m.providerInitializing = true
		return m, nil

	case CopilotStatusMsg:
		if msg.Info != nil && msg.Info.Error == "" && msg.Info.Username != "" {
			// Already authenticated — show a toast with the account info and
			// open the model picker pre-filtered to copilot so the user can pick a model.
			planStr := ""
			if msg.Info.Plan != "" {
				planStr = " · " + msg.Info.Plan
			}
			cmds = append(cmds, m.showToast(
				fmt.Sprintf("Copilot: logged in as %s%s", msg.Info.Username, planStr),
				ToastInfo, 6*time.Second,
			))
			// Switch to the model dialog pre-filtered to copilot
			return m.openModelDialogFiltered("copilot")
		}
		// Not authenticated or error — show the viewport info
		m.viewport.SetContent(m.renderCopilotUsageView(msg.Info))
		m.setStatus("Copilot status fetched — press Esc or send a message to return")
		return m, nil

	case ModelsRefreshedMsg:
		// Rebuild the model list in-place if the model dialog is open
		if m.view == ViewModels {
			m.modelList = m.buildModelList()
		}
		return m, nil

	case ProviderChangedMsg:
		m.providerInitializing = false
		m.providerInitError = nil
		m.Provider = msg.Provider
		m.Model_ = msg.Model
		if msg.Engine != nil {
			m.Engine = msg.Engine
		}
		m.setStatus(fmt.Sprintf("Switched to %s / %s", msg.Provider, msg.Model))
		return m, nil

	case AnthropicOAuthReadyMsg:
		if msg.Err != nil {
			m.setStatus("OAuth error: " + msg.Err.Error())
			if m.oauthDialog != nil {
				m.oauthDialog.SetOAuthError("Failed to generate URL: " + msg.Err.Error())
			}
			return m, nil
		}
		// Update the dialog with the real URL and remember the verifier
		m.oauthVerifier = msg.Verifier
		if m.oauthDialog != nil {
			m.oauthDialog.OAuthURL = msg.URL
		}
		// Try to open in browser (best-effort)
		go func() { _ = config.OpenBrowser(msg.URL) }()
		return m, nil

	case AnthropicOAuthDoneMsg:
		m.view = m.previousView
		m.oauthDialog = nil
		m.oauthVerifier = ""
		m.focusTextarea()
		if msg.Err != nil {
			m.setStatus("OAuth failed: " + msg.Err.Error())
		} else {
			m.setStatus("Anthropic OAuth login successful")
			// Re-init engine with new OAuth credentials
			return m, m.reinitEngine()
		}
		return m, nil

	case CopilotDeviceCodeMsg:
		if msg.Err != nil {
			if m.oauthDialog != nil {
				m.oauthDialog.SetOAuthError("Failed to start device flow: " + msg.Err.Error())
			}
			return m, nil
		}
		// Update dialog with the user code and verification URL
		if m.oauthDialog != nil {
			m.oauthDialog.OAuthURL = fmt.Sprintf("%s  •  Code: %s", msg.VerificationURI, msg.UserCode)
		}
		// Try to open browser
		go func() { _ = config.OpenBrowser(msg.VerificationURI) }()
		// Start polling for the token in background
		deviceCode := msg.DeviceCode
		intervalSecs := msg.Interval
		expiresIn := msg.ExpiresIn
		return m, func() tea.Msg {
			err := provider.PollCopilotDeviceFlow(deviceCode, intervalSecs, expiresIn)
			return CopilotLoginDoneMsg{Err: err}
		}

	case CopilotLoginDoneMsg:
		if m.oauthDialog != nil {
			m.view = m.previousView
			m.oauthDialog = nil
			m.focusTextarea()
		}
		if msg.Err != nil {
			m.setStatus("Copilot login failed: " + msg.Err.Error())
			return m, m.showToast("Copilot login failed: "+msg.Err.Error(), ToastError, 8*time.Second)
		}
		m.setStatus("GitHub Copilot login successful")
		toastCmd := m.showToast("✓ GitHub Copilot authenticated!", ToastInfo, 5*time.Second)
		// Open the model dialog filtered to copilot so the user can pick a model
		m2, dialogCmd := m.openModelDialogFiltered("copilot")
		return m2, tea.Batch(toastCmd, dialogCmd)

	case LogoutDoneMsg:
		if msg.Err != nil {
			m.setStatus("Logout failed: " + msg.Err.Error())
			return m, m.showToast("Logout failed: "+msg.Err.Error(), ToastError, 5*time.Second)
		}
		m.setStatus("Logged out from " + msg.Provider)
		// If we logged out from the current provider, we need the engine to fail gracefully
		if msg.Provider == m.Provider {
			m.Engine = nil
		}
		return m, m.showToast("✓ Logged out from "+msg.Provider, ToastInfo, 4*time.Second)

	case OAuthCodeSubmitMsg:
		// Exchange the code for tokens
		if msg.Code != "" && m.oauthVerifier != "" {
			verifier := m.oauthVerifier
			code := msg.Code
			return m, func() tea.Msg {
				token, err := provider.AnthropicOAuthExchange(code, verifier)
				if err != nil {
					return AnthropicOAuthDoneMsg{Err: err}
				}
				if err := provider.SaveAnthropicOAuthToken(token); err != nil {
					return AnthropicOAuthDoneMsg{Err: err}
				}
				return AnthropicOAuthDoneMsg{}
			}
		}
		// No code or verifier — just close the dialog
		m.view = m.previousView
		m.oauthDialog = nil
		m.oauthVerifier = ""
		m.focusTextarea()
		return m, nil

	case APIKeySubmitMsg:
		// Save the API key for the given provider and reinit the engine
		m.view = m.previousView
		m.oauthDialog = nil
		m.focusTextarea()
		if msg.Key == "" {
			return m, nil
		}
		provName := msg.Provider
		apiKey := msg.Key
		m.Provider = provName
		m.Engine = nil // force reinit with new key
		return m, func() tea.Msg {
			creds, _ := config.LoadCredentials()
			if creds == nil {
				creds = &config.Credentials{}
			}
			config.SetProviderKey(creds, provName, apiKey)
			if err := config.SaveCredentials(creds); err != nil {
				return ErrorMsg{Error: fmt.Errorf("failed to save key: %w", err)}
			}
			_ = config.SaveDefaultProvider(provName)
			return ProviderChangedMsg{Provider: provName, Model: m.Model_}
		}

	case RevertDoneMsg:
		// Reload messages from store after revert
		if m.sessionID != "" {
			if sess, err := m.Store.Get(m.sessionID); err == nil {
				m.messages = sess.Messages
			}
		}
		m.updateViewport()
		m.setStatus("Reverted last change (files restored)")
		return m, nil

	case UndoDoneMsg:
		return m.handleUndoDoneMsg(msg)

	case RedoDoneMsg:
		return m.handleRedoDoneMsg(msg)

	case SnapshotCapturedMsg:
		m.handleSnapshotCaptured(msg)
		return m, nil

	case CompactDoneMsg:
		// Reload messages after compaction
		if m.sessionID != "" {
			if sess, err := m.Store.Get(m.sessionID); err == nil {
				m.messages = sess.Messages
			}
		}
		m.updateViewport()
		m.setStatus("Session compacted successfully")
		return m, nil

	case ExternalEditorDoneMsg:
		if msg.Err != nil {
			m.setStatus("Editor error: " + msg.Err.Error())
		} else {
			m.textarea.SetValue(msg.Content)
			m.textarea.CursorEnd()
			m.focusTextarea()
		}
		return m, nil

	case ToastDismissMsg:
		m.pruneToasts()
		// If there are still live toasts, schedule another check
		if len(m.toasts) > 0 {
			nearest := m.toasts[0].Expiry
			for _, t := range m.toasts[1:] {
				if t.Expiry.Before(nearest) {
					nearest = t.Expiry
				}
			}
			remaining := time.Until(nearest)
			if remaining < 50*time.Millisecond {
				remaining = 50 * time.Millisecond
			}
			cmds = append(cmds, toastTickCmd(remaining+100*time.Millisecond))
		}

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
	key := msg.String()

	// ── Global shortcuts that always apply ────────────────────────────
	switch key {
	case "tab":
		if m.focusInput {
			m.blurTextarea()
		} else {
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
		// If there are code blocks, enter copy-select mode (next digit = block index)
		if len(m.lastCodeBlocks) > 0 {
			if len(m.lastCodeBlocks) == 1 {
				// Only one block — copy it immediately
				return m.copyCodeBlock(1)
			}
			m.copyPending = true
			m.setStatus(fmt.Sprintf("Copy code block: press 1-%d (or Esc to cancel)", len(m.lastCodeBlocks)))
			return m, nil
		}
		return m.copyLastMessage()
	case "ctrl+m":
		// Toggle mouse capture. Off by default so text selection works normally.
		m.mouseEnabled = !m.mouseEnabled
		if m.mouseEnabled {
			m.setStatus("Mouse enabled — hold Shift to select text, Ctrl+M to disable")
			return m, tea.EnableMouseCellMotion
		}
		m.setStatus("Mouse disabled — select text freely, Ctrl+M to enable mouse")
		return m, tea.DisableMouse
	case "ctrl+k":
		m.blurTextarea()
		return m.openModelDialog()
	case "ctrl+j":
		return m.cycleAgent(1)
	case "ctrl+p":
		m.blurTextarea()
		return m.openCommandPalette()
	case "ctrl+shift+p":
		m.blurTextarea()
		return m.openProviderDialog()
	case "ctrl+s":
		m.blurTextarea()
		return m.openSettings()
	case "ctrl+n":
		return m, m.createSession()
	case "ctrl+z":
		// Undo last AI file change
		return m.undoLastChange()
	case "ctrl+shift+z":
		// Redo last undone AI file change
		return m.redoLastChange()
	case "ctrl+e":
		// Open current textarea content in $EDITOR
		return m.openExternalEditor()
	case "ctrl+l":
		m.previousView = m.view
		m.view = ViewSessions
		m.sessionList = m.Store.List()
		m.selectedSession = 0
		return m, nil
	case "ctrl+shift+l":
		m.messages = []session.Message{}
		m.lastWelcomeContent = ""
		if m.tokenTracker != nil {
			m.tokenTracker.Reset()
		}
		m.updateViewport()
		m.setStatus("Screen cleared")
		return m, nil
	}

	// ── Viewport focused (chat history scroll) ────────────────────────
	if !m.focusInput {
		consumed := m.handleVimViewport(key)
		if consumed {
			return m, nil
		}
		// Fall back to bubbletea viewport for any unhandled keys
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	// ── Input focused – Esc: mode switch or cancel ────────────────────
	if key == "esc" {
		if m.vimState.Mode == VimModeInsert {
			// Switch INSERT → NORMAL (move cursor back one, like real vim)
			m.vimState.Mode = VimModeNormal
			m.vimState.reset()
			col := m.textarea.LineInfo().CharOffset
			if col > 0 {
				m.textarea.SetCursor(col - 1)
			}
			m.setStatus("-- NORMAL --")
			return m, nil
		}
		// Already in NORMAL: cancel streaming or clear pending ops
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
		}
		m.vimState.reset()
		return m, nil
	}

	// ── Input focused – Vim NORMAL mode ──────────────────────────────
	if m.vimState.Mode == VimModeNormal {
		consumed, switchToInsert := m.handleVimNormal(key)
		if switchToInsert {
			m.vimState.Mode = VimModeInsert
			m.vimState.reset()
			m.setStatus("-- INSERT --")
		}
		if consumed {
			return m, nil
		}
	}

	// ── Permission prompt – absorb all keys when active ───────────────
	if handled, cmd := m.updatePermission(key); handled {
		return m, cmd
	}

	// ── Question prompt – absorb all keys when active ─────────────────
	if handled, cmd := m.updateQuestion(key); handled {
		return m, cmd
	}

	// ── Copy-pending mode: next digit selects which code block to copy ──
	if m.copyPending {
		if key == "esc" {
			m.copyPending = false
			m.setStatus("Copy cancelled")
			return m, nil
		}
		if len(key) == 1 && key[0] >= '1' && key[0] <= '9' {
			idx := int(key[0] - '0')
			m.copyPending = false
			return m.copyCodeBlock(idx)
		}
		// Any other key cancels copy-pending mode
		m.copyPending = false
	}

	// ── Input focused – Vim INSERT mode (normal typing) ───────────────

	// Check autocomplete first
	if m.handleAutocompleteKey(key) {
		return m, nil
	}

	switch key {
	case "up":
		// History navigation: go to previous entry when in INSERT mode
		if m.vimState.Mode == VimModeInsert && m.promptHistory != nil {
			prev := m.promptHistory.Up(m.textarea.Value())
			m.textarea.SetValue(prev)
			m.textarea.CursorEnd()
			return m, nil
		}
	case "down":
		// History navigation: go to next entry / back to blank
		if m.vimState.Mode == VimModeInsert && m.promptHistory != nil {
			next := m.promptHistory.Down()
			m.textarea.SetValue(next)
			m.textarea.CursorEnd()
			return m, nil
		}
	case "enter":
		if m.isStreaming {
			return m, nil
		}
		input := strings.TrimSpace(m.textarea.Value())
		if input == "" {
			return m, nil
		}
		// Persist to history
		if m.promptHistory != nil {
			m.promptHistory.Append(input)
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
	}

	// Forward to textarea for normal character input
	if !m.isStreaming {
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		// Check if autocomplete should open/update
		m.maybeOpenAutocomplete()
		return m, cmd
	}
	return m, nil
}

// copyCodeBlock copies the Nth code block (1-based) from lastCodeBlocks.
func (m Model) copyCodeBlock(idx int) (tea.Model, tea.Cmd) {
	if idx < 1 || idx > len(m.lastCodeBlocks) {
		m.setStatus(fmt.Sprintf("No code block #%d (found %d)", idx, len(m.lastCodeBlocks)))
		return m, nil
	}
	code := m.lastCodeBlocks[idx-1]
	if err := clipboard.WriteAll(code); err != nil {
		m.setStatus("Failed to copy: " + err.Error())
		return m, nil
	}
	lines := strings.Count(code, "\n") + 1
	m.setStatus(fmt.Sprintf("Copied code block #%d  (%d lines, %d chars)", idx, lines, len(code)))
	return m, nil
}

// copyLastMessage copies the last assistant message text to clipboard.
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

// providerStaticList returns the canonical ordered list of providers with display metadata.
// Priority 0 = "Popular", 99 = "Other".
func providerStaticList() []ProviderInfo {
	return []ProviderInfo{
		{Name: "anthropic", DisplayName: "Anthropic", Description: "Claude Sonnet, Opus, Haiku", EnvVar: []string{"ANTHROPIC_API_KEY"}, Priority: 1},
		{Name: "copilot", DisplayName: "GitHub Copilot", Description: "Claude, GPT-4.1, Gemini via GitHub", EnvVar: []string{"GITHUB_TOKEN"}, Priority: 2},
		{Name: "openai", DisplayName: "OpenAI", Description: "GPT-4.1, o3, o4-mini", EnvVar: []string{"OPENAI_API_KEY"}, Priority: 3},
		{Name: "google", DisplayName: "Google Gemini", Description: "Gemini 2.5 Pro/Flash, 2.0 Flash", EnvVar: []string{"GOOGLE_API_KEY", "GEMINI_API_KEY"}, Priority: 4},
		{Name: "xai", DisplayName: "xAI", Description: "Grok 4, Grok 3, Grok Code", EnvVar: []string{"XAI_API_KEY"}, Priority: 5},
		{Name: "groq", DisplayName: "Groq", Description: "Llama, Qwen, Kimi (ultra-fast)", EnvVar: []string{"GROQ_API_KEY"}, Priority: 6},
		{Name: "deepseek", DisplayName: "DeepSeek", Description: "DeepSeek Chat, Reasoner", EnvVar: []string{"DEEPSEEK_API_KEY"}, Priority: 7},
		{Name: "openrouter", DisplayName: "OpenRouter", Description: "Multi-provider gateway (200+ models)", EnvVar: []string{"OPENROUTER_API_KEY"}, Priority: 8},
		{Name: "mistral", DisplayName: "Mistral", Description: "Devstral, Magistral, Codestral", EnvVar: []string{"MISTRAL_API_KEY"}, Priority: 99},
		{Name: "bedrock", DisplayName: "Amazon Bedrock", Description: "Claude, Llama via AWS", EnvVar: []string{"AWS_ACCESS_KEY_ID"}, Priority: 99},
		{Name: "azure", DisplayName: "Azure OpenAI", Description: "OpenAI models via Azure", EnvVar: []string{"AZURE_OPENAI_API_KEY"}, Priority: 99},
		{Name: "google-vertex", DisplayName: "Google Vertex AI", Description: "Gemini models via GCP", EnvVar: []string{"GOOGLE_APPLICATION_CREDENTIALS"}, Priority: 99},
		{Name: "deepinfra", DisplayName: "DeepInfra", Description: "GLM, Kimi, DeepSeek (fast inference)", EnvVar: []string{"DEEPINFRA_API_KEY"}, Priority: 99},
		{Name: "cerebras", DisplayName: "Cerebras", Description: "Qwen, GPT-OSS, GLM (ultra-fast)", EnvVar: []string{"CEREBRAS_API_KEY"}, Priority: 99},
		{Name: "together", DisplayName: "Together AI", Description: "Open models (fast inference)", EnvVar: []string{"TOGETHER_API_KEY"}, Priority: 99},
		{Name: "cloudflare-workers-ai", DisplayName: "Cloudflare Workers AI", Description: "GPT-OSS, Llama, Qwen on Edge", EnvVar: []string{"CLOUDFLARE_API_TOKEN"}, Priority: 99},
		{Name: "gitlab", DisplayName: "GitLab Duo", Description: "Claude, GPT via GitLab", EnvVar: []string{"GITLAB_TOKEN"}, Priority: 99},
		{Name: "sambanova", DisplayName: "SambaNova", Description: "Llama, DeepSeek (fast)", EnvVar: []string{"SAMBANOVA_API_KEY"}, Priority: 99},
		{Name: "fireworks", DisplayName: "Fireworks AI", Description: "Llama, Qwen, Mixtral", EnvVar: []string{"FIREWORKS_API_KEY"}, Priority: 99},
		{Name: "huggingface", DisplayName: "Hugging Face", Description: "Open models via Inference API", EnvVar: []string{"HUGGINGFACE_API_KEY", "HF_TOKEN"}, Priority: 99},
		{Name: "replicate", DisplayName: "Replicate", Description: "Open models via Replicate", EnvVar: []string{"REPLICATE_API_TOKEN"}, Priority: 99},
		{Name: "perplexity", DisplayName: "Perplexity", Description: "Search-augmented LLMs", EnvVar: []string{"PERPLEXITY_API_KEY"}, Priority: 99},
		{Name: "cohere", DisplayName: "Cohere", Description: "Command R+ for enterprise", EnvVar: []string{"COHERE_API_KEY"}, Priority: 99},
	}
}

func (m *Model) buildProviderList() []ProviderInfo {
	available := m.Config.ListAvailableProviders()
	connectedSet := make(map[string]bool)
	for _, p := range available {
		connectedSet[p] = true
	}
	all := providerStaticList()
	for i := range all {
		all[i].Connected = connectedSet[all[i].Name]
	}
	return all
}

func (m *Model) providerDialogLen() int { return len(m.providerList) }

// updateProviderDialog handles keys in the provider list, including the
// action sub-menu (Select Model / Logout) shown for connected providers.
func (m Model) updateProviderDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.providerActionMode {
		switch msg.String() {
		case "esc":
			m.providerActionMode = false
			return m, nil
		case "up", "k":
			if m.providerActionIdx > 0 {
				m.providerActionIdx--
			}
			return m, nil
		case "down", "j":
			if m.providerActionIdx < 1 {
				m.providerActionIdx++
			}
			return m, nil
		case "enter":
			return m.execProviderAction()
		}
		return m, nil
	}
	// Direct logout shortcut: press 'l' on a connected provider row
	if msg.String() == "l" && m.dialogSelected < len(m.providerList) {
		if sel := m.providerList[m.dialogSelected]; sel.Connected {
			m.providerActionIdx = 1 // Logout action
			return m.execProviderAction()
		}
	}
	return m.updateDialog(msg, m.providerDialogLen(), m.onProviderSelect)
}

func (m Model) onProviderSelect() (tea.Model, tea.Cmd) {
	if m.dialogSelected >= len(m.providerList) {
		return m, nil
	}
	sel := m.providerList[m.dialogSelected]

	if !sel.Connected {
		switch sel.Name {
		case "anthropic":
			return m.openOAuthCodeDialog("Claude Pro/Max", "", "Paste the authorization code here:")
		case "copilot":
			return m.openCopilotDeviceFlowDialog()
		default:
			urlHint := ""
			if len(sel.EnvVar) > 0 {
				urlHint = "env: " + sel.EnvVar[0]
			}
			return m.openAPIKeyDialog(sel.DisplayName, urlHint, sel.Name)
		}
	}

	// Connected: enter action sub-menu (Select Model / Logout)
	m.providerActionMode = true
	m.providerActionIdx = 0
	return m, nil
}

// execProviderAction handles the action sub-menu for a connected provider.
func (m Model) execProviderAction() (tea.Model, tea.Cmd) {
	if m.dialogSelected >= len(m.providerList) {
		return m, nil
	}
	sel := m.providerList[m.dialogSelected]
	m.providerActionMode = false

	switch m.providerActionIdx {
	case 0: // Select Model
		return m.openModelDialogFiltered(sel.Name)
	case 1: // Logout
		m.view = m.previousView
		m.focusTextarea()
		provName := sel.Name
		m.setStatus("Logging out from " + provName + "...")
		return m, func() tea.Msg {
			creds, _ := config.LoadCredentials()
			if creds == nil {
				creds = &config.Credentials{}
			}
			config.ClearProviderCredential(creds, provName)
			_ = config.SaveCredentials(creds)
			if provName == "copilot" {
				config.RemoveCopilotOAuthToken()
			}
			if provName == "anthropic" {
				provider.ClearAnthropicOAuthToken()
			}
			return LogoutDoneMsg{Provider: provName}
		}
	}
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

// openModelDialogFiltered opens the model dialog pre-filtered to a specific provider.
// When called from the provider dialog (m.view == ViewProviders), we intentionally
// keep the existing previousView so that confirming a model returns to ViewChat, not ViewProviders.
func (m Model) openModelDialogFiltered(provName string) (tea.Model, tea.Cmd) {
	if m.view != ViewProviders {
		m.previousView = m.view
	}
	m.view = ViewModels
	m.dialogSelected = 0
	m.dialogFilter = provName
	m.modelList = m.buildModelList()
	return m, nil
}

func (m *Model) buildModelList() []ModelItem {
	var models []ModelItem

	// Build a set of recent keys for fast lookup
	recentSet := make(map[string]int) // "provider/id" → position (0=most recent)
	for i, r := range m.recentModels {
		recentSet[r] = i
	}

	available := m.Config.ListAvailableProviders()
	connectedSet := make(map[string]bool)
	for _, p := range available {
		connectedSet[p] = true
	}

	// Helper: get display name for a provider
	provDisplay := func(id string) string {
		for _, p := range providerStaticList() {
			if p.Name == id {
				return p.DisplayName
			}
		}
		return id
	}

	// Iterate over ALL providers in the registry (not just connected)
	// so we can show models even when switching to a provider
	for _, provID := range m.modelRegistry.ListProviders() {
		if !connectedSet[provID] {
			continue // only show models for connected providers
		}
		infoList := m.modelRegistry.ListModels(provID)
		pName := provDisplay(provID)
		for _, info := range infoList {
			key := provID + "/" + info.ID
			_, recent := recentSet[key]
			models = append(models, ModelItem{
				ID:           info.ID,
				Name:         info.Name,
				Provider:     provID,
				ProviderName: pName,
				Context:      info.Limits.Context,
				Selected:     info.ID == m.Model_ && provID == m.Provider,
				IsFree:       info.Cost.Input == 0 && info.Cost.Output == 0,
				CostInput:    info.Cost.Input,
				CostOutput:   info.Cost.Output,
				HasReasoning: info.Capabilities.Reasoning,
				HasVision:    info.Capabilities.Input.Image,
				IsRecent:     recent,
			})
		}
	}

	// Fallback: if registry has no models for a connected provider, use prov.Models().
	// For Copilot specifically, always fetch the live /models list from the API so we
	// show only the models that are actually enabled on this account (free vs paid).
	for _, provName := range available {
		regModels := m.modelRegistry.ListModels(provName)

		// For Copilot: merge live API model list with registry metadata
		if provName == "copilot" {
			copilotProv, err := provider.NewCopilotProvider()
			if err == nil {
				// Build a lookup of registry models by ID for metadata enrichment
				regByID := make(map[string]provider.ModelInfo, len(regModels))
				for _, rm := range regModels {
					regByID[rm.ID] = rm
				}
				liveIDs, fetchErr := copilotProv.FetchModels()
				if fetchErr == nil && len(liveIDs) > 0 {
					pName := provDisplay(provName)
					for _, modelID := range liveIDs {
						key := provName + "/" + modelID
						_, recent := recentSet[key]
						mi := ModelItem{
							ID:           modelID,
							Name:         modelID,
							Provider:     provName,
							ProviderName: pName,
							Selected:     modelID == m.Model_ && provName == m.Provider,
							IsRecent:     recent,
							IsFree:       true, // Copilot subscription covers cost
						}
						// Enrich with registry metadata if available
						if rm, ok := regByID[modelID]; ok {
							mi.Name = rm.Name
							mi.Context = rm.Limits.Context
							mi.HasReasoning = rm.Capabilities.Reasoning
							mi.HasVision = rm.Capabilities.Input.Image
						}
						models = append(models, mi)
					}
					continue // skip the generic fallback below for copilot
				}
			}
			// If live fetch fails, fall through to generic fallback
		}

		if len(regModels) > 0 {
			continue // already populated from registry above
		}
		apiKey, _ := config.GetAPIKeyWithFallback(provName, m.Config)
		prov, err := provider.CreateProvider(provName, apiKey)
		if err != nil {
			continue
		}
		pName := provDisplay(provName)
		for _, modelID := range prov.Models() {
			key := provName + "/" + modelID
			_, recent := recentSet[key]
			models = append(models, ModelItem{
				ID:           modelID,
				Name:         modelID,
				Provider:     provName,
				ProviderName: pName,
				Selected:     modelID == m.Model_ && provName == m.Provider,
				IsRecent:     recent,
			})
		}
	}

	// Sort: current provider first, then alphabetical by provider then model name
	sort.Slice(models, func(i, j int) bool {
		pi, pj := models[i].Provider, models[j].Provider
		if pi == m.Provider && pj != m.Provider {
			return true
		}
		if pi != m.Provider && pj == m.Provider {
			return false
		}
		if pi != pj {
			return pi < pj
		}
		ni := models[i].Name
		if ni == "" {
			ni = models[i].ID
		}
		nj := models[j].Name
		if nj == "" {
			nj = models[j].ID
		}
		return ni < nj
	})
	return models
}

// addRecentModel prepends "provider/modelID" to the recents list (max 5, deduplicated).
func (m *Model) addRecentModel(provID, modelID string) {
	key := provID + "/" + modelID
	// Remove existing entry if present
	out := make([]string, 0, len(m.recentModels))
	for _, r := range m.recentModels {
		if r != key {
			out = append(out, r)
		}
	}
	// Prepend and cap at 5
	out = append([]string{key}, out...)
	if len(out) > 5 {
		out = out[:5]
	}
	m.recentModels = out
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
			strings.Contains(strings.ToLower(mi.Provider), f) ||
			strings.Contains(strings.ToLower(mi.ProviderName), f) {
			out = append(out, mi)
		}
	}
	return out
}

func (m Model) updateModelDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	filtered := m.filteredModels()
	displayLen := len(m.modelDisplayOrder)
	if displayLen == 0 {
		// Fallback: if display order hasn't been built yet, use filtered length
		displayLen = len(filtered)
	}
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
		if m.dialogSelected < displayLen-1 {
			m.dialogSelected++
		}
		return m, nil
	case "enter":
		// Map dialogSelected (display position) → filtered index
		filteredIdx := m.dialogSelected
		if m.dialogSelected < len(m.modelDisplayOrder) {
			filteredIdx = m.modelDisplayOrder[m.dialogSelected]
		}
		if filteredIdx < len(filtered) {
			sel := filtered[filteredIdx]
			oldProvider := m.Provider
			m.Provider = sel.Provider
			m.Model_ = sel.ID
			m.addRecentModel(sel.Provider, sel.ID)
			m.view = m.previousView
			m.dialogFilter = ""
			m.focusTextarea()
			name := sel.Name
			if name == "" {
				name = sel.ID
			}
			m.setStatus(fmt.Sprintf("Model: %s (%s)", name, sel.ProviderName))
			// Rebuild engine if provider changed (model change within same provider
			// takes effect on next message through the engine's model field)
			if sel.Provider != oldProvider {
				m.Engine = nil
				m.providerInitializing = true
				m.providerInitError = nil
				return m, m.reinitEngine()
			}
			// Update engine model even for same-provider model switches
			if m.Engine != nil {
				m.Engine.SetModel(sel.ID)
			}
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
	// Update engine agent live
	if m.Engine != nil {
		ag := agent.GetAgent(m.Agent, m.Config)
		m.Engine.SetAgent(ag)
	}
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
		// Update engine agent live
		if m.Engine != nil {
			ag := agent.GetAgent(m.Agent, m.Config)
			m.Engine.SetAgent(ag)
		}
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
	case "theme.change":
		return m.openThemeDialog()
	case "help":
		m.view = ViewHelp
		return m, nil
	case "compact":
		if m.sessionID != "" && m.Engine != nil {
			m.setStatus("Compacting session...")
			return m, func() tea.Msg {
				if err := m.Engine.CompactSession(context.Background(), m.sessionID); err != nil {
					return ErrorMsg{Error: fmt.Errorf("compact failed: %w", err)}
				}
				return CompactDoneMsg{}
			}
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
	case "undo":
		return m.undoLastChange()
	case "redo":
		return m.redoLastChange()
	case "copy":
		if len(m.lastCodeBlocks) > 0 {
			return m.copyCodeBlock(len(m.lastCodeBlocks))
		}
		return m.copyLastMessage()
	case "clear":
		m.messages = []session.Message{}
		m.lastWelcomeContent = ""
		m.updateViewport()
		return m, nil
	case "quit":
		return m, tea.Quit
	case "login":
		return m.openOAuthCodeDialog("Claude Pro/Max", "", "Paste the authorization code here:")
	case "copilot.login":
		return m.openCopilotCommandPaletteFlow()
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
	// ── Rename input mode ────────────────────────────────────────────
	if m.renamingSession {
		switch msg.String() {
		case "esc":
			m.renamingSession = false
			m.renameBuffer = ""
		case "enter":
			if m.renameBuffer != "" && m.selectedSession < len(m.sessionList) {
				_ = m.Store.UpdateTitle(m.sessionList[m.selectedSession].ID, m.renameBuffer)
				m.sessionList = m.Store.List()
			}
			m.renamingSession = false
			m.renameBuffer = ""
		case "backspace":
			if len(m.renameBuffer) > 0 {
				m.renameBuffer = m.renameBuffer[:len(m.renameBuffer)-1]
			}
		default:
			if k := msg.String(); len(k) == 1 {
				m.renameBuffer += k
			}
		}
		return m, nil
	}

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
	case "r":
		// Start inline rename for selected session
		if m.selectedSession < len(m.sessionList) {
			m.renamingSession = true
			m.renameBuffer = m.sessionList[m.selectedSession].Title
		}
		return m, nil
	case "f":
		// Fork the selected session (copy all messages into a new session)
		if m.selectedSession < len(m.sessionList) {
			src := m.sessionList[m.selectedSession]
			forked, err := m.Store.Fork(src.ID, -1) // -1 = copy all messages
			if err != nil {
				m.setStatus("Fork error: " + err.Error())
			} else {
				m.sessionList = m.Store.List()
				m.setStatus(fmt.Sprintf("Forked → %s", forked.Title))
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
		// Capture snapshot BEFORE file-modifying tools so we can undo them.
		// Batch with the stream-wait cmd so both run concurrently.
		fileModifyingTools := map[string]bool{
			"write": true, "edit": true, "patch": true,
			"apply_patch": true, "bash": true,
		}
		if fileModifyingTools[msg.Event.ToolName] && m.streamCh != nil {
			detail := msg.Event.Content
			if detail == "" {
				detail = msg.Event.ToolName
			}
			return m, tea.Batch(waitForStream(m.streamCh), m.captureSnapshotCmd(detail))
		}
	case "tool_end":
		m.currentTool = ""
		// Mark matching tool as completed and attach any diff data
		for i := len(m.streamingTools) - 1; i >= 0; i-- {
			if m.streamingTools[i].Name == msg.Event.ToolName && m.streamingTools[i].Active {
				m.streamingTools[i].Active = false
				// Capture diff data for realtime display in the streaming viewport
				if msg.Event.DiffData != nil {
					m.streamingTools[i].Diffs = append(m.streamingTools[i].Diffs, msg.Event.DiffData)
				}
				for _, d := range msg.Event.DiffDataList {
					m.streamingTools[i].Diffs = append(m.streamingTools[i].Diffs, d)
				}
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
		return m, m.showToast("Error: "+msg.Event.Content, ToastError, 6*time.Second)
	case "done":
		m.isStreaming = false
		m.viewport.Height = m.calcViewportHeight()
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
	m.viewport.Height = m.calcViewportHeight()
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

// applyThemeToComponents re-creates syntax/markdown/diff components using the
// current theme. Called after any theme change.
func (m *Model) applyThemeToComponents() {
	m.syntaxHighlighter = components.NewSyntaxHighlighter(m.currentTheme.SyntaxTheme)
	m.markdownRenderer, _ = components.NewMarkdownRenderer(m.width-4, m.currentTheme.MarkdownTheme)
	m.diffViewer = components.NewDiffViewer(m.width, m.currentTheme.SyntaxTheme)
}

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
		m.lastWelcomeContent = ""
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
		if m.sessionID != "" && m.Engine != nil {
			m.setStatus("Compacting session...")
			return m, func() tea.Msg {
				if err := m.Engine.CompactSession(context.Background(), m.sessionID); err != nil {
					return ErrorMsg{Error: fmt.Errorf("compact failed: %w", err)}
				}
				return CompactDoneMsg{}
			}
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
	case "/copilot-usage", "/copilot":
		// Show Copilot authentication status and session token usage
		if m.Provider != "copilot" {
			m.setStatus("Not using Copilot provider (current: " + m.Provider + ")")
			return m, nil
		}
		m.setStatus("Fetching Copilot status...")
		return m, fetchCopilotStatusCmd()
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
			return m, nil
		}
		// Open theme picker dialog
		return m.openThemeDialog()
	case "/login", "/auth":
		// Show OAuth code dialog for Claude Pro/Max
		return m.openOAuthCodeDialog(
			"Claude Pro/Max",
			"", // URL is generated dynamically via PKCE
			"Paste the authorization code here:",
		)
	case "/undo":
		return m.undoLastChange()
	case "/redo":
		return m.redoLastChange()
	case "/copy":
		// /copy N  — copy Nth code block from last response
		if len(parts) > 1 {
			idx := 0
			fmt.Sscanf(parts[1], "%d", &idx)
			if idx > 0 {
				return m.copyCodeBlock(idx)
			}
		}
		// No index — copy last code block or full message
		if len(m.lastCodeBlocks) > 0 {
			return m.copyCodeBlock(len(m.lastCodeBlocks))
		}
		return m.copyLastMessage()
	case "/copilot-login":
		// GitHub Copilot device-flow login
		return m.openCopilotDeviceFlowDialog()
	case "/logout":
		// Log out from the current provider
		provName := m.Provider
		if len(parts) > 1 {
			provName = parts[1]
		}
		provName = strings.TrimSpace(provName)
		if provName == "" {
			m.setStatus("Usage: /logout [provider] — e.g. /logout anthropic")
			return m, nil
		}
		m.setStatus("Logging out from " + provName + "...")
		return m, func() tea.Msg {
			creds, _ := config.LoadCredentials()
			if creds == nil {
				creds = &config.Credentials{}
			}
			config.ClearProviderCredential(creds, provName)
			if err := config.SaveCredentials(creds); err != nil {
				return LogoutDoneMsg{Provider: provName, Err: err}
			}
			// Also remove copilot device OAuth token if logging out copilot
			if provName == "copilot" {
				config.RemoveCopilotOAuthToken()
			}
			// Also remove anthropic OAuth token if logging out anthropic
			if provName == "anthropic" {
				provider.ClearAnthropicOAuthToken()
			}
			return LogoutDoneMsg{Provider: provName}
		}
	default:
		// Check custom slash commands from .dcode/commands/ config
		cmdName := strings.TrimPrefix(cmd, "/")
		if m.Config.Commands != nil {
			if customCmd, ok := m.Config.Commands[cmdName]; ok {
				// Expand template with any args
				tmpl := customCmd.Template
				if len(parts) > 1 {
					args := strings.Join(parts[1:], " ")
					tmpl = strings.ReplaceAll(tmpl, "{{args}}", args)
					tmpl = strings.ReplaceAll(tmpl, "{{input}}", args)
				}
				if tmpl == "" {
					m.setStatus("Custom command /" + cmdName + " has no template")
					return m, nil
				}
				m.setStatus("Running custom command: /" + cmdName)
				return m, m.sendMessage(tmpl)
			}
		}
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
	case ViewOAuthCode:
		return m.renderOverlay(m.renderOAuthCodeDialog())
	case ViewTheme:
		return m.renderOverlay(m.renderThemeDialog())
	case ViewSessions:
		return m.renderSessionListView()
	case ViewHelp:
		return m.renderHelpView()
	default:
		return m.injectToastsIntoView(m.renderChat())
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

	// ── Build the main chat column ──────────────────────────────────────────
	var chat strings.Builder

	// Header (extracted to header.go)
	chat.WriteString(m.renderHeader())

	// Welcome / viewport content
	if len(m.messages) == 0 && !m.isStreaming {
		t := m.currentTheme
		dim := lipgloss.NewStyle().Foreground(t.TextMuted)
		acc := lipgloss.NewStyle().Foreground(t.Primary)
		accDim := lipgloss.NewStyle().Foreground(t.Primary).Faint(true)

		welcome := "\n"

		// Use full ASCII logo for wide terminals, compact text for narrow ones
		if m.viewport.Width >= 48 {
			welcome += acc.Bold(true).Render(`  ██████╗  ██████╗ ██████╗ ██████╗ ███████╗`) + "\n"
			welcome += acc.Bold(true).Render(`  ██╔══██╗██╔════╝██╔═══██╗██╔══██╗██╔════╝`) + "\n"
			welcome += acc.Render(`  ██║  ██║██║     ██║   ██║██║  ██║█████╗  `) + "\n"
			welcome += accDim.Render(`  ██║  ██║██║     ██║   ██║██║  ██║██╔══╝  `) + "\n"
			welcome += accDim.Render(`  ██████╔╝╚██████╗╚██████╔╝██████╔╝███████╗`) + "\n"
			welcome += dim.Render(`  ╚═════╝  ╚═════╝ ╚═════╝ ╚═════╝ ╚══════╝`) + "\n"
			welcome += dim.Render("  AI-powered coding assistant") + "\n"
		} else {
			// Compact single-line branding for narrow terminals
			welcome += acc.Bold(true).Render("  DCode") + "  " + dim.Render("AI coding assistant") + "\n"
		}

		if m.providerInitError != nil {
			errSt := lipgloss.NewStyle().Foreground(t.Error)
			welcome += "\n" + errSt.Render("  ⚠ "+m.providerInitError.Error()) + "\n"
			welcome += dim.Render("  Run `dcode auth login` to set up authentication.") + "\n"
		}

		if welcome != m.lastWelcomeContent {
			m.lastWelcomeContent = welcome
			m.viewport.SetContent(welcome)
		}
	}

	// Render viewport with scrollbar
	chat.WriteString(m.RenderViewportWithScrollbar())
	chat.WriteString("\n")

	// Loading / streaming indicator — always shown when active or streaming
	if m.loadingState.IsActive {
		chat.WriteString(m.RenderLoadingState() + "\n")
	} else if m.isStreaming {
		// Fallback indicator when streaming but no specific loading state
		chat.WriteString(m.spinner.View() + " " + dimStyle.Render("Generating response") + "\n")
	}

	// Permission prompt (rendered above the input area when active)
	if perm := m.renderPermissionPrompt(); perm != "" {
		chat.WriteString(perm + "\n")
	}

	// Question prompt (rendered above the input area when active)
	if q := m.renderQuestionPrompt(); q != "" {
		chat.WriteString(q + "\n")
	}

	// Input separator + textarea (with autocomplete popup above)
	chatW := m.viewport.Width + 2
	if ac := m.renderAutocomplete(); ac != "" {
		chat.WriteString(ac + "\n")
	}
	chat.WriteString(dimStyle.Render(strings.Repeat("─", chatW)) + "\n")
	chat.WriteString(m.textarea.View())
	chat.WriteString("\n")

	// Footer (extracted to footer.go)
	chat.WriteString(m.renderFooter())

	// ── Compose chat + sidebar side-by-side ────────────────────────────────
	// Only show sidebar when terminal is wide enough
	if m.width >= 80+sidebarWidth {
		return lipgloss.JoinHorizontal(lipgloss.Top, chat.String(), m.renderSidebar())
	}
	return chat.String()
}

// ─── Provider dialog ────────────────────────────────────────────────────────────

func (m *Model) renderProviderDialog() string {
	t := m.currentTheme
	dim := lipgloss.NewStyle().Foreground(t.TextMuted)
	sectionSt := lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
	selSt := lipgloss.NewStyle().Foreground(t.Background).Background(t.Primary).Bold(true)
	normalSt := lipgloss.NewStyle().Foreground(t.Text)
	connDot := lipgloss.NewStyle().Foreground(t.Success)
	dimDot := lipgloss.NewStyle().Foreground(t.TextMuted)
	curMark := lipgloss.NewStyle().Foreground(t.Accent)
	descSt := lipgloss.NewStyle().Foreground(t.TextMuted)
	envSt := lipgloss.NewStyle().Foreground(t.TextDim)

	w := clampWidth(m.width, 62)

	var b strings.Builder
	titleSt := lipgloss.NewStyle().Bold(true).Foreground(t.Primary).Padding(0, 1)
	b.WriteString(titleSt.Render("Select Provider") + "\n")
	b.WriteString(dim.Render(strings.Repeat("─", w)) + "\n")

	logoutSt := lipgloss.NewStyle().Foreground(red)
	logoutBadgeSt := lipgloss.NewStyle().Foreground(red).Faint(true)
	loggedInSt := lipgloss.NewStyle().Foreground(t.Success).Faint(true)
	actionSelSt := lipgloss.NewStyle().Foreground(t.Background).Background(red).Bold(true)

	writeRow := func(p ProviderInfo, idx int) {
		cursor := "  "
		nameSt := normalSt
		if idx == m.dialogSelected {
			cursor = "▸ "
			nameSt = selSt
		}
		dot := dimDot.Render("○ ")
		if p.Connected {
			dot = connDot.Render("● ")
		}
		active := ""
		if p.Name == m.Provider {
			active = curMark.Render(" ✓")
		}
		envHint := ""
		if !p.Connected && len(p.EnvVar) > 0 {
			envHint = "  " + envSt.Render(p.EnvVar[0])
		}
		// For connected providers: show "Logged in" badge and [l] logout hint
		loggedInBadge := ""
		logoutHint := ""
		if p.Connected {
			loggedInBadge = "  " + loggedInSt.Render("Logged in")
			logoutHint = "  " + logoutBadgeSt.Render("[l] logout")
		}
		b.WriteString(fmt.Sprintf("  %s%s%s%s%s  %s%s%s\n",
			cursor, dot, nameSt.Render(p.DisplayName), active,
			loggedInBadge, descSt.Render(p.Description), envHint, logoutHint))

		// Inline action sub-menu for the selected connected provider (Select Model only)
		if idx == m.dialogSelected && m.providerActionMode && p.Connected {
			actions := []string{"Select Model", "Logout"}
			for ai, action := range actions {
				aCursor := "      "
				aSt := normalSt
				if ai == m.providerActionIdx {
					aCursor = "    ▸ "
					if ai == 1 {
						aSt = actionSelSt
					} else {
						aSt = selSt
					}
				} else if ai == 1 {
					aSt = logoutSt
				}
				b.WriteString(fmt.Sprintf("%s%s\n", aCursor, aSt.Render(action)))
			}
		}
	}

	// Popular providers (priority < 99)
	hasPopular := false
	for i, p := range m.providerList {
		if p.Priority >= 99 {
			continue
		}
		if !hasPopular {
			b.WriteString("\n" + sectionSt.Render("  Popular") + "\n")
			hasPopular = true
		}
		writeRow(p, i)
	}

	// Other providers
	hasOther := false
	for i, p := range m.providerList {
		if p.Priority < 99 {
			continue
		}
		if !hasOther {
			b.WriteString("\n" + dim.Render("  Other") + "\n")
			hasOther = true
		}
		writeRow(p, i)
	}

	if m.providerActionMode {
		b.WriteString("\n" + dim.Render("  ↑/↓ navigate  Enter: select  Esc: back"))
	} else {
		b.WriteString("\n" + dim.Render("  ↑/↓ navigate  Enter: select  l: logout  Esc: cancel"))
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.BorderHighlight).
		Padding(0, 1).
		Width(w).
		Render(b.String())
}

// ─── Model dialog ───────────────────────────────────────────────────────────────

func (m *Model) renderModelDialog() string {
	t := m.currentTheme
	dim := lipgloss.NewStyle().Foreground(t.TextMuted)
	dimDim := lipgloss.NewStyle().Foreground(t.TextDim)
	titleSt := lipgloss.NewStyle().Bold(true).Foreground(t.Primary).Padding(0, 1)
	sectionSt := lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
	selSt := lipgloss.NewStyle().Foreground(t.Background).Background(t.Primary).Bold(true)
	normalSt := lipgloss.NewStyle().Foreground(t.Text)
	freeSt := lipgloss.NewStyle().Foreground(t.Success)
	checkSt := lipgloss.NewStyle().Foreground(t.Accent)
	reasonSt := lipgloss.NewStyle().Foreground(t.Secondary)
	visionSt := lipgloss.NewStyle().Foreground(t.Info)

	w := clampWidth(m.width, 72)
	filtered := m.filteredModels()

	var b strings.Builder

	// Title + search bar
	b.WriteString(titleSt.Render("Select Model") + "\n")
	filterPrompt := dim.Render("  Search: ")
	if m.dialogFilter != "" {
		filterPrompt += lipgloss.NewStyle().Foreground(t.Accent).Render(m.dialogFilter)
	} else {
		filterPrompt += dim.Render("type to filter...")
	}
	b.WriteString(filterPrompt + "\n")
	b.WriteString(dim.Render(strings.Repeat("─", w)) + "\n")

	if len(filtered) == 0 {
		b.WriteString(dim.Render("\n  No models found for connected providers.\n"))
		b.WriteString(dim.Render("  Run `dcode login` to connect a provider.\n"))
	} else {
		// Build display list: when no filter, recents float to the top.
		// Each entry: (originalIndex into filtered, sectionHeader string or "")
		type displayEntry struct {
			idx     int
			section string // non-empty means emit a section header before this row
		}
		var display []displayEntry

		if m.dialogFilter == "" {
			// Pass 1: recents section
			hasRecents := false
			for i, mi := range filtered {
				if !mi.IsRecent {
					continue
				}
				if !hasRecents {
					display = append(display, displayEntry{idx: i, section: "Recent"})
					hasRecents = true
				} else {
					display = append(display, displayEntry{idx: i})
				}
			}
			// Pass 2: all non-recent models, grouped by provider
			lastProv := ""
			for i, mi := range filtered {
				if mi.IsRecent {
					continue
				}
				prov := mi.ProviderName
				if prov == "" {
					prov = strings.ToUpper(mi.Provider)
				}
				if prov != lastProv {
					display = append(display, displayEntry{idx: i, section: prov})
					lastProv = prov
				} else {
					display = append(display, displayEntry{idx: i})
				}
			}
		} else {
			// When filtering, just show flat list (no section grouping)
			for i := range filtered {
				display = append(display, displayEntry{idx: i})
			}
		}

		// Store the display→filtered index mapping so updateModelDialog can use it
		m.modelDisplayOrder = make([]int, len(display))
		for di, de := range display {
			m.modelDisplayOrder[di] = de.idx
		}

		// Scrolling window: show maxShow rows centered around dialogSelected
		maxShow := 18
		start := m.dialogSelected - maxShow/2
		if start < 0 {
			start = 0
		}
		if start+maxShow > len(display) {
			start = len(display) - maxShow
			if start < 0 {
				start = 0
			}
		}
		end := start + maxShow
		if end > len(display) {
			end = len(display)
		}

		renderRow := func(displayPos int, mi ModelItem) {
			cursor := "    "
			nameSt := normalSt
			if displayPos == m.dialogSelected {
				cursor = "  ▸ "
				nameSt = selSt
			}
			disp := mi.Name
			if disp == "" {
				disp = mi.ID
			}
			var badges []string
			if mi.Selected {
				badges = append(badges, checkSt.Render("✓"))
			}
			if mi.IsFree {
				badges = append(badges, freeSt.Render("free"))
			} else if mi.CostInput > 0 {
				badges = append(badges, dimDim.Render(fmt.Sprintf("$%.2f/M", mi.CostInput)))
			}
			if mi.HasReasoning {
				badges = append(badges, reasonSt.Render("think"))
			}
			if mi.HasVision {
				badges = append(badges, visionSt.Render("vision"))
			}
			if mi.Context > 0 {
				badges = append(badges, dim.Render(fmt.Sprintf("%dk", mi.Context/1000)))
			}
			badgeStr := ""
			if len(badges) > 0 {
				badgeStr = "  " + strings.Join(badges, " ")
			}
			b.WriteString(fmt.Sprintf("%s%s%s\n", cursor, nameSt.Render(disp), badgeStr))
		}

		for displayPos, de := range display {
			if displayPos < start || displayPos >= end {
				continue
			}
			if de.section != "" {
				sSt := sectionSt
				if de.section == "Recent" {
					sSt = lipgloss.NewStyle().Foreground(t.Warning).Bold(true)
				}
				b.WriteString("\n  " + sSt.Render(de.section) + "\n")
			}
			renderRow(displayPos, filtered[de.idx])
		}

		// Scroll indicator
		if len(display) > maxShow {
			b.WriteString("\n" + dim.Render(fmt.Sprintf(
				"  %d/%d  ↑/↓ to scroll", m.dialogSelected+1, len(display),
			)) + "\n")
		}
	}

	b.WriteString("\n" + dim.Render("  Enter: select  Type: filter  Esc: cancel"))
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.BorderHighlight).
		Padding(0, 1).
		Width(w).
		Render(b.String())
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

// ─── OAuth Code dialog ──────────────────────────────────────────────────────────

func (m *Model) renderOAuthCodeDialog() string {
	if m.oauthDialog == nil {
		return ""
	}
	return m.oauthDialog.View()
}

// OAuthCodeSubmitMsg is sent when user submits an authorization code
type OAuthCodeSubmitMsg struct {
	Code string
}

// APIKeySubmitMsg is sent when user submits an API key for a provider
type APIKeySubmitMsg struct {
	Provider string
	Key      string
}

// AnthropicOAuthReadyMsg carries the dynamically-generated OAuth URL + PKCE verifier
type AnthropicOAuthReadyMsg struct {
	URL      string
	Verifier string
	Err      error
}

// AnthropicOAuthDoneMsg is sent after successful token exchange
type AnthropicOAuthDoneMsg struct {
	Err error
}

// CopilotDeviceCodeMsg is sent when the GitHub device code has been obtained
type CopilotDeviceCodeMsg struct {
	UserCode        string
	VerificationURI string
	DeviceCode      string
	Interval        int
	ExpiresIn       int
	Err             error
}

// CopilotLoginDoneMsg is sent after copilot device flow completes
type CopilotLoginDoneMsg struct {
	Err error
}

// CopilotDeviceFlowPollMsg is a tick to poll for copilot token
type CopilotDeviceFlowPollMsg struct{}

// LogoutDoneMsg is sent after a provider is logged out
type LogoutDoneMsg struct {
	Provider string
	Err      error
}

// openOAuthCodeDialog opens the OAuth dialog. For Anthropic it first generates a real
// PKCE URL; a temporary "Generating..." URL is shown while the async Cmd runs.
func (m Model) openOAuthCodeDialog(title, staticURL, instruction string) (tea.Model, tea.Cmd) {
	m.previousView = m.view
	m.view = ViewOAuthCode

	// Show dialog immediately with a placeholder URL; it will be updated once the
	// async PKCE generation completes.
	displayURL := staticURL
	if title == "Claude Pro/Max" {
		displayURL = "Generating authorization URL..."
	}

	m.oauthDialog = components.NewOAuthCodeDialog(title, displayURL, instruction, func(code string) tea.Msg {
		return OAuthCodeSubmitMsg{Code: code}
	})
	m.oauthDialog.Show()

	var cmd tea.Cmd
	if title == "Claude Pro/Max" {
		cmd = func() tea.Msg {
			result, err := provider.AnthropicOAuthAuthorize()
			if err != nil {
				return AnthropicOAuthReadyMsg{Err: err}
			}
			return AnthropicOAuthReadyMsg{URL: result.URL, Verifier: result.Verifier}
		}
	}
	return m, cmd
}

// updateOAuthCodeDialog handles input in the OAuth code dialog
func (m Model) updateOAuthCodeDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.oauthDialog == nil {
		m.view = m.previousView
		return m, nil
	}

	// Let the dialog handle the key
	dialog, cmd := m.oauthDialog.Update(msg)
	m.oauthDialog = dialog

	// Check if dialog was closed
	if !m.oauthDialog.IsVisible() {
		m.view = m.previousView
		m.focusTextarea()
	}

	return m, cmd
}

// openAPIKeyDialog opens a dialog for the user to enter an API key for provName.
// It reuses the OAuth code dialog infrastructure — Enter submits, Esc cancels.
func (m Model) openAPIKeyDialog(displayName, urlHint, provName string) (tea.Model, tea.Cmd) {
	m.previousView = m.view
	m.view = ViewOAuthCode
	instruction := "Paste your API key and press Enter  (Esc to cancel)"
	if urlHint != "" {
		instruction = urlHint + "  •  " + instruction
	}
	m.oauthDialog = components.NewOAuthCodeDialog(displayName, "", instruction, func(key string) tea.Msg {
		return APIKeySubmitMsg{Provider: provName, Key: key}
	})
	m.oauthDialog.Show()
	return m, nil
}

// openCopilotCommandPaletteFlow handles the "GitHub Copilot — OAuth Login" command palette entry.
// If the user is already logged in, it shows the Copilot model list instead of starting a new flow.
// If not logged in (or the token is missing), it opens the device-flow OAuth dialog.
func (m Model) openCopilotCommandPaletteFlow() (tea.Model, tea.Cmd) {
	// Check if a Copilot OAuth token already exists
	copilotProv, err := provider.NewCopilotProvider()
	if err == nil && copilotProv != nil {
		// Already have a token — show status and open model selection for Copilot
		m.setStatus("Copilot: already logged in — fetching status...")
		return m, tea.Batch(
			fetchCopilotStatusCmd(),
			func() tea.Msg {
				// Open model dialog filtered to copilot after status fetch
				return nil
			},
		)
	}
	// Not logged in — start device flow
	return m.openCopilotDeviceFlowDialog()
}

// openCopilotDeviceFlowDialog starts the GitHub device OAuth flow for Copilot,
// showing the user code and verification URL in a familiar OAuth dialog.
func (m Model) openCopilotDeviceFlowDialog() (tea.Model, tea.Cmd) {
	m.previousView = m.view
	m.view = ViewOAuthCode

	// Show an immediate placeholder while the device code is being fetched
	m.oauthDialog = components.NewOAuthCodeDialog(
		"GitHub Copilot",
		"Requesting device code...",
		"Open the URL below in your browser — we'll detect authorization automatically",
		func(_ string) tea.Msg {
			return CopilotLoginDoneMsg{}
		},
	)
	m.oauthDialog.Show()

	// Start device code request in background
	cmd := func() tea.Msg {
		result, err := provider.StartCopilotDeviceFlow()
		if err != nil {
			return CopilotDeviceCodeMsg{Err: err}
		}
		r, ok := result.(*provider.CopilotDeviceCodeResponse)
		if !ok || r == nil {
			return CopilotDeviceCodeMsg{Err: fmt.Errorf("unexpected response type from device flow")}
		}
		return CopilotDeviceCodeMsg{
			DeviceCode:      r.DeviceCode,
			UserCode:        r.UserCode,
			VerificationURI: r.VerificationURI,
			Interval:        r.Interval,
			ExpiresIn:       r.ExpiresIn,
		}
	}
	return m, cmd
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
		// If we're renaming this entry, show the buffer instead
		if i == m.selectedSession && m.renamingSession {
			title = m.renameBuffer + "█"
			style = lipgloss.NewStyle().Foreground(yellow).Bold(true)
		}
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

	hint := "Enter: select | R: rename | F: fork | D: delete | Esc: back"
	if m.renamingSession {
		hint = "Enter: confirm rename | Esc: cancel"
	}
	b.WriteString("\n" + dimStyle.Render(hint))
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
		{"Ctrl+E", "Open textarea in $EDITOR"},
		{"Ctrl+Y", "Copy last code block (then digit to pick: 1-9)"},
		{"Ctrl+M", "Toggle mouse (disable to select text)"},
		{"Ctrl+K", "Select model"},
		{"Ctrl+J", "Cycle agent forward"},
		{"Ctrl+P", "Command palette"},
		{"Ctrl+Shift+P", "Select provider"},
		{"Ctrl+N", "New session"},
		{"Ctrl+L", "Toggle sessions"},
		{"Ctrl+Shift+L", "Clear screen"},
		{"Ctrl+S", "Settings"},
		{"Ctrl+Z", "Undo last AI file change"},
		{"Ctrl+Shift+Z", "Redo last undone change"},
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
		{"/login or /auth", "OAuth login (Claude Pro/Max auth code input)"},
		{"/new", "Create new session"},
		{"/session list", "List all sessions"},
		{"/copy [N]", "Copy code block #N from last response"},
		{"/undo", "Undo last AI file change"},
		{"/redo", "Redo last undone change"},
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
		{"Ctrl+Y", "Copy last code block (press digit 1-9 to pick specific block)"},
		{"Ctrl+M", "Toggle mouse off so you can select & copy text freely"},
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
	contentWidth := m.calcContentWidth()

	for idx, msg := range m.messages {
		switch msg.Role {
		case "user":
			m.renderUserMessage(&content, msg, idx, contentWidth)
		case "assistant":
			m.renderAssistantMessage(&content, msg, contentWidth)
		}
	}

	// Streaming content — text in bordered block, tools as plain lines below
	if m.isStreaming {
		// 1. Thinking block — rendered OUTSIDE text border, like opencode
		if m.streamingThinking.Len() > 0 {
			content.WriteString(m.renderThinkingBlock(m.streamingThinking.String(), contentWidth, true) + "\n")
		}

		var textBlock strings.Builder

		// 2. Retry info — inside text border
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
			textBlock.WriteString(retryStyle.Render(
				fmt.Sprintf("⟳ %s (attempt %d, retrying in %s)",
					retryMsg, m.retryInfo.Attempt, formatDuration(remaining)),
			) + "\n\n")
		}

		// 3. Streaming text with cursor — inside text border
		if m.streamingText.Len() > 0 {
			textBlock.WriteString(m.streamingText.String() + dimStyle.Render("▊"))
		}

		if textBlock.Len() > 0 {
			bordered := assistantBorderStyle.Width(contentWidth).Render(textBlock.String())
			content.WriteString("\n" + bordered + "\n")
		}

		// 4. Tool calls — rendered as plain compact lines OUTSIDE the border
		for _, tc := range m.streamingTools {
			content.WriteString(m.renderToolLine(tc.Name, tc.Detail, tc.Active) + "\n")
			// Inline diffs below the tool line
			for _, dd := range tc.Diffs {
				if m.diffViewer != nil {
					diffView := m.diffViewer.RenderEditDiff(dd.OldContent, dd.NewContent, dd.FilePath, 30)
					content.WriteString(diffView + "\n")
				}
			}
		}
	}

	m.viewport.SetContent(content.String())
	m.viewport.GotoBottom()
}

// calcContentWidth returns the viewport content width accounting for borders/padding.
func (m *Model) calcContentWidth() int {
	w := m.viewport.Width - 4 // 1 char left border + 2 padding + 1 slack
	if w < 20 {
		w = 20
	}
	return w
}

// extractThinkingTopic tries to find the first **bold** phrase in the thinking
// text and returns it as a short topic label (mirrors opencode behaviour).
func extractThinkingTopic(text string) string {
	// Look for **...** pattern
	start := strings.Index(text, "**")
	if start == -1 {
		return ""
	}
	end := strings.Index(text[start+2:], "**")
	if end == -1 {
		return ""
	}
	topic := strings.TrimSpace(text[start+2 : start+2+end])
	if len(topic) > 60 {
		topic = topic[:57] + "..."
	}
	return topic
}

// renderThinkingBlock renders an opencode-style thinking panel.
//   - streaming=true  → shows live text with spinner and "Thinking..." header
//   - streaming=false → shows a compact collapsed summary (topic + char count)
func (m *Model) renderThinkingBlock(text string, width int, streaming bool) string {
	t := m.currentTheme

	thinkColor := lipgloss.Color("#B4A8FF") // soft lavender for thinking
	dimColor := lipgloss.NewStyle().Foreground(t.TextMuted)
	thinkSt := lipgloss.NewStyle().Foreground(thinkColor)
	thinkDimSt := lipgloss.NewStyle().Foreground(thinkColor).Faint(true).Italic(true)

	topic := extractThinkingTopic(text)

	if !streaming {
		// Completed: show a single collapsed line with topic + char count
		icon := thinkSt.Render("◉")
		label := thinkSt.Render("Thought")
		detail := ""
		if topic != "" {
			detail = dimColor.Render(" about ") + thinkDimSt.Render(topic)
		}
		charCount := dimColor.Render(fmt.Sprintf("  ·  %d chars", len(text)))
		line := "  " + icon + " " + label + detail + charCount
		return line
	}

	// Streaming: show animated header + last N lines of thinking text
	var b strings.Builder

	// Header line: spinner + "Thinking" + topic
	spinnerView := m.spinner.View()
	headerLabel := thinkSt.Bold(true).Render("Thinking")
	topicStr := ""
	if topic != "" {
		topicStr = dimColor.Render(" about ") + thinkDimSt.Render(topic)
	}
	b.WriteString("  " + spinnerView + " " + headerLabel + topicStr + "\n")

	// Show the last ~8 lines of the thinking text (tail, like a live log)
	const maxThinkLines = 8
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if len(lines) > maxThinkLines {
		hiddenCount := len(lines) - maxThinkLines
		lines = lines[len(lines)-maxThinkLines:]
		b.WriteString("  " + dimColor.Render(fmt.Sprintf("  (%d lines above)", hiddenCount)) + "\n")
	}
	for _, line := range lines {
		// Truncate very long lines
		if lipgloss.Width(line) > width-6 {
			line = line[:width-9] + "..."
		}
		b.WriteString("  " + thinkDimSt.Render(line) + "\n")
	}

	// Wrap in a left-border block styled distinctly from assistant messages
	thinkBorder := lipgloss.NewStyle().
		Border(lipgloss.Border{Left: "┃"}).
		BorderForeground(thinkColor).
		PaddingLeft(1).
		Width(width)
	return thinkBorder.Render(b.String())
}

// renderToolLine renders a single tool call line in OpenCode style:
//
//	→ ls ✓ Running ls...
func (m *Model) renderToolLine(name, detail string, active bool) string {
	icon, clr := getToolIcon(name)
	iconSt := lipgloss.NewStyle().Foreground(clr)
	var status string
	if active {
		status = m.spinner.View()
	} else {
		status = successStyle.Render("✓")
	}
	line := "  " + iconSt.Render(icon) + " " + highlightStyle.Render(name) + " " + status
	if detail != "" {
		line += " " + dimStyle.Render(detail)
	}
	return line
}

func (m *Model) renderUserMessage(b *strings.Builder, msg session.Message, idx int, width int) {
	// Skip pure tool-result messages (no visible user text)
	if msg.Content == "" {
		hasVisible := false
		for _, p := range msg.Parts {
			if p.Type != "tool_result" {
				hasVisible = true
				break
			}
		}
		if !hasVisible {
			// Still render tool result diffs if present
			for _, part := range msg.Parts {
				if part.Type == "tool_result" && part.Metadata != nil && m.diffViewer != nil {
					diffs := extractDiffDataFromMetadata(part.Metadata)
					for _, dd := range diffs {
						diffView := m.diffViewer.RenderEditDiff(dd.OldContent, dd.NewContent, dd.FilePath, 30)
						b.WriteString(diffView + "\n")
					}
				}
			}
			return
		}
	}

	var inner strings.Builder

	if msg.Content != "" {
		inner.WriteString(userMsgStyle.Render(msg.Content))
	}

	// Render image attachment indicators
	for _, part := range msg.Parts {
		if part.Type == "image" && part.Image != nil {
			name := part.Image.FileName
			if name == "" {
				name = part.Image.MediaType
			}
			inner.WriteString("\n" + dimStyle.Render("[img] "+name))
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
	// ── Text content ────────────────────────────────────────────────────────
	// Prose and code are rendered as separate blocks so that code boxes
	// (which contain raw ANSI lines) are never passed through a lipgloss
	// Width()-constrained Render() call that would collapse them.
	if msg.Content != "" {
		m.renderAssistantTextBlocks(b, msg.Content, width)
	}

	// ── Reasoning block (completed) ─────────────────────────────────────────
	for _, part := range msg.Parts {
		if part.Type == "reasoning" && part.Content != "" {
			b.WriteString(m.renderThinkingBlock(part.Content, width, false) + "\n")
		}
	}

	// ── Tool call lines (OpenCode style: compact, outside border) ──────────
	// Collect all tool_use parts and render as a group
	var toolLines []string
	for _, part := range msg.Parts {
		if part.Type != "tool_use" {
			continue
		}
		detail := formatToolInput(part.ToolName, part.ToolInput)
		active := part.Status == "running" || part.Status == "pending"
		line := m.renderToolLine(part.ToolName, detail, active)
		// Override status for error
		if part.Status == "error" {
			icon, clr := getToolIcon(part.ToolName)
			iconSt := lipgloss.NewStyle().Foreground(clr)
			line = "  " + iconSt.Render(icon) + " " + highlightStyle.Render(part.ToolName) +
				" " + errorStyle.Render("✗")
			if detail != "" {
				line += " " + dimStyle.Render(detail)
			}
		}
		toolLines = append(toolLines, line)
	}
	for _, tl := range toolLines {
		b.WriteString(tl + "\n")
	}

	// ── Error parts ─────────────────────────────────────────────────────────
	for _, part := range msg.Parts {
		if part.Type == "error" && part.Content != "" {
			errContent := errorStyle.Render("Error: " + part.Content)
			bordered := assistantBorderStyle.Width(width).Render(errContent)
			b.WriteString(bordered + "\n")
		}
	}

	// ── Completion footer ────────────────────────────────────────────────────
	// Show footer if there is any content (text or tool calls)
	hasContent := msg.Content != ""
	if !hasContent {
		for _, p := range msg.Parts {
			if p.Type == "tool_use" || p.Type == "error" {
				hasContent = true
				break
			}
		}
	}
	if hasContent {
		agentName := msg.AgentName
		if agentName == "" {
			agentName = m.Agent
		}
		modelID := msg.ModelID
		if modelID == "" {
			modelID = m.Model_
		}
		footer := dimStyle.Render("  ") + lipgloss.NewStyle().Foreground(purple).Render("▣") + " " +
			lipgloss.NewStyle().Foreground(txtClr).Render(agentName)
		if modelID != "" {
			footer += dimStyle.Render(" · " + shortModel(modelID))
		}
		if msg.TokensIn > 0 || msg.TokensOut > 0 {
			footer += dimStyle.Render(fmt.Sprintf(" · %d↑ %d↓", msg.TokensIn, msg.TokensOut))
		}
		b.WriteString(footer + "\n")
	}
}

// splitTextSegments parses markdown text into alternating prose / code segments.
type textSegment struct {
	kind string // "prose" | "code"
	text string
	lang string
}

func splitTextSegments(text string) []textSegment {
	var segs []textSegment
	var proseLines, codeLines []string
	inBlock := false
	codeLang := ""

	for _, line := range strings.Split(text, "\n") {
		if !inBlock && strings.HasPrefix(line, "```") {
			if len(proseLines) > 0 {
				segs = append(segs, textSegment{kind: "prose", text: strings.Join(proseLines, "\n")})
				proseLines = nil
			}
			inBlock = true
			codeLang = strings.TrimSpace(strings.TrimPrefix(line, "```"))
			codeLines = nil
		} else if inBlock && strings.TrimSpace(line) == "```" {
			segs = append(segs, textSegment{kind: "code", text: strings.Join(codeLines, "\n"), lang: codeLang})
			inBlock = false
			codeLines = nil
			codeLang = ""
		} else if inBlock {
			codeLines = append(codeLines, line)
		} else {
			proseLines = append(proseLines, line)
		}
	}
	// flush
	if inBlock && len(codeLines) > 0 {
		segs = append(segs, textSegment{kind: "code", text: strings.Join(codeLines, "\n"), lang: codeLang})
	} else if len(proseLines) > 0 {
		segs = append(segs, textSegment{kind: "prose", text: strings.Join(proseLines, "\n")})
	}
	return segs
}

// renderAssistantTextBlocks writes prose and code segments directly to b.
//
// Key design: prose segments go inside assistantBorderStyle (purple ┃ left border).
// Code blocks are written OUTSIDE any Width()-constrained Render() call so
// that the hand-built box lines (which contain raw ANSI) are never reflowed.
//
//	\n
//	┃  prose paragraph …         ← assistantBorderStyle.Width(width).Render(prose)
//	┃  more prose …
//	\n
//	╭─ python ──────── ctrl+y → 1 ─╮   ← renderCodeBlock(), written raw
//	│ def foo():
//	╰──────────────────────────────╯
//	\n
//	┃  continuation prose …
func (m *Model) renderAssistantTextBlocks(b *strings.Builder, text string, width int) {
	segs := splitTextSegments(text)

	// assistantBorderStyle left-border overhead: ┃ (1) + PaddingLeft(2) = 3
	// so the inner prose width is width - 3.
	proseInnerW := width - 3
	if proseInnerW < 20 {
		proseInnerW = 20
	}

	// Code box width: we want it to match the outer width (same as the
	// assistantBorderStyle outer box). Subtract 0 — it's rendered at the
	// same nesting level as the border box (both are direct children of the
	// viewport content string).
	codeBoxW := width
	if codeBoxW < 24 {
		codeBoxW = 24
	}

	// Reset code block list for this message render
	m.lastCodeBlocks = m.lastCodeBlocks[:0]

	// Accumulate consecutive prose segments so they share one border block.
	var proseBuf strings.Builder
	codeIdx := 0

	flushProse := func() {
		prose := strings.TrimSpace(proseBuf.String())
		proseBuf.Reset()
		if prose == "" {
			return
		}
		var rendered string
		if m.markdownRenderer != nil {
			if r, err := m.markdownRenderer.Render(prose); err == nil {
				r = strings.TrimRight(r, "\n")
				r = strings.TrimLeft(r, "\n")
				rendered = r
			}
		}
		if rendered == "" {
			rendered = assistantMsgStyle.Render(prose)
		}
		// Wrap prose in the coloured left-border block
		block := assistantBorderStyle.Width(width).Render(rendered)
		b.WriteString("\n" + block + "\n")
	}

	for _, seg := range segs {
		switch seg.kind {
		case "prose":
			proseBuf.WriteString(seg.text + "\n")
		case "code":
			// Flush any accumulated prose first
			flushProse()
			codeIdx++
			m.lastCodeBlocks = append(m.lastCodeBlocks, seg.text)
			b.WriteString(m.renderCodeBlock(seg.text, seg.lang, codeIdx, codeBoxW) + "\n")
		}
	}
	// Flush remaining prose
	flushProse()
}

// renderAssistantText is kept for streaming use (still returns a string).
func (m *Model) renderAssistantText(text string, width int) string {
	var b strings.Builder
	m.renderAssistantTextBlocks(&b, text, width)
	return strings.TrimRight(b.String(), "\n")
}

// renderCodeBlock renders a single code block with proper line-by-line syntax
// highlighting. The border is built manually so that ANSI-escaped code lines
// are never passed through a lipgloss Width()-constrained Render() call
// (which would collapse all lines into one).
//
// The box anatomy and char counts (all plain, no ANSI):
//
//	╭─ python ──────────────────── ctrl+y → 1 ─╮
//	│  <ansi line>                              ← no right border (unknown ANSI width)
//	╰────────────────────────────────────────────╯
//
// Top row char count:
//
//	"╭" + "─" + " " + lang + " " + fill + " " + hint + " " + "─" + "╮"
//	  1  +  1  +  1  + lW  +  1  + fW  +  1  + hW  +  1  +  1  +  1   = width
//	fixed = 8   → fillLen = width - 8 - lW - hW
//
// Bottom row: "╰" + (width-2)×"─" + "╯" = width chars. ✓
func (m *Model) renderCodeBlock(code, lang string, idx, width int) string {
	const minWidth = 32
	if width < minWidth {
		width = minWidth
	}

	langLabel := lang
	if langLabel == "" {
		langLabel = "code"
	}

	// ── Syntax-highlight ────────────────────────────────────────────────────
	var hlLines []string
	if m.syntaxHighlighter != nil {
		hlLines = m.syntaxHighlighter.HighlightLines(code, lang)
	} else {
		hlLines = strings.Split(code, "\n")
	}
	// Strip trailing blank lines chroma appends
	for len(hlLines) > 0 && strings.TrimSpace(hlLines[len(hlLines)-1]) == "" {
		hlLines = hlLines[:len(hlLines)-1]
	}

	// ── Plain-text widths (lipgloss.Width strips ANSI) ──────────────────────
	lbl := lipgloss.NewStyle().Foreground(purple).Bold(true).Render(langLabel)
	hint := lipgloss.NewStyle().Foreground(overlay).Faint(true).
		Render(fmt.Sprintf("ctrl+y → %d", idx))
	lW := lipgloss.Width(lbl)
	hW := lipgloss.Width(hint)

	// fixed = ╭(1) ─(1) space(1) + space(1) fill space(1) + space(1) hint space(1) ─(1) ╮(1) = 9
	// Simplified: "╭─ " = 3, " " = 1, " " = 1, " ─╮" = 3  → total fixed = 8
	const fixed = 8
	fillLen := width - fixed - lW - hW
	if fillLen < 1 {
		fillLen = 1
	}

	bdr := func(s string) string { return lipgloss.NewStyle().Foreground(surface).Render(s) }

	// ╭─ python ──────────── ctrl+y → 1 ─╮
	top := bdr("╭─ ") + lbl + bdr(" "+strings.Repeat("─", fillLen)+" ") + hint + bdr(" ─╮")
	// ╰──────────────────────────────────╯  (width chars total)
	bot := bdr("╰" + strings.Repeat("─", width-2) + "╯")

	var b strings.Builder
	b.WriteString(top + "\n")
	for _, hl := range hlLines {
		b.WriteString(bdr("│") + "  " + hl + "\n")
	}
	b.WriteString(bot)
	return b.String()
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
		// Sync cfg.Provider/Model so the engine uses the correct defaults.
		m.Config.Provider = m.Provider
		if m.Model_ != "" {
			m.Config.Model = m.Model_
		}
		m.Engine = session.NewPromptEngine(m.Store, prov, m.Config, ag, registry)
		m.providerInitializing = false
		m.providerInitError = nil
	}

	m.isStreaming = true
	m.viewport.Height = m.calcViewportHeight()
	m.streamingText.Reset()
	m.streamingThinking.Reset()
	m.streamingTools = nil
	m.retryInfo = nil
	m.lastCodeBlocks = m.lastCodeBlocks[:0] // reset code blocks for new response
	m.copyPending = false

	// Snapshot pending images and clear them before any async work
	pendingImgs := m.pendingImages
	m.pendingImages = nil

	localMsg := session.Message{Role: "user", Content: input}
	for _, img := range pendingImgs {
		imgCopy := img
		localMsg.Parts = append(localMsg.Parts, session.Part{Type: "image", Image: &imgCopy})
	}
	m.messages = append(m.messages, localMsg)
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

	// Wire the interactive permission callback so the engine can pause and
	// wait for the user to approve or reject a tool call in the TUI.
	m.Engine.OnPermissionAsk(func(toolName, description string) bool {
		replyCh := make(chan bool, 1)
		req := &permission.Request{
			Action:      permission.Action(toolName),
			Description: description,
			Path:        toolName,
		}
		ch <- PermissionRequestMsg{Req: req, ReplyCh: replyCh}
		// Block until the TUI resolves the permission
		select {
		case allowed := <-replyCh:
			return allowed
		case <-ctx.Done():
			return false
		}
	})

	// Wire the interactive question callback so the question tool shows the TUI dialog.
	m.Engine.OnQuestionAsk(func(items []tool.QuestionItem) [][]string {
		// Convert tool.QuestionItem → tui.QuestionItem
		tuiItems := make([]QuestionItem, len(items))
		for i, it := range items {
			opts := make([]QuestionOption, len(it.Options))
			for j, o := range it.Options {
				opts[j] = QuestionOption{Label: o.Label, Description: o.Description}
			}
			tuiItems[i] = QuestionItem{
				Header:   it.Header,
				Question: it.Question,
				Options:  opts,
				Multiple: it.Multiple,
				Custom:   it.Custom,
			}
		}
		replyCh := make(chan [][]string, 1)
		req := QuestionRequest{Questions: tuiItems}
		ch <- QuestionRequestMsg{Req: req, ReplyCh: replyCh}
		// Block until the TUI resolves the question (or context cancelled)
		select {
		case answers := <-replyCh:
			return answers
		case <-ctx.Done():
			return nil
		}
	})

	go func() {
		defer close(ch)
		err := m.Engine.RunWithAttachments(ctx, m.sessionID, input, pendingImgs)
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

// revertLastStep undoes the most recent AI step by reverting the files it changed.
func (m Model) revertLastStep() (tea.Model, tea.Cmd) {
	if m.sessionID == "" {
		m.setStatus("No active session to revert")
		return m, nil
	}
	if m.isStreaming {
		m.setStatus("Cannot revert while generating")
		return m, nil
	}

	sess, err := m.Store.Get(m.sessionID)
	if err != nil {
		m.setStatus("Revert error: " + err.Error())
		return m, nil
	}

	// Find the last assistant message that has a patch record
	var targetMsgID string
	for i := len(sess.Messages) - 1; i >= 0; i-- {
		msg := sess.Messages[i]
		if msg.Role == "system" {
			for _, part := range msg.Parts {
				if part.Type == "patch" && part.PatchHash != "" {
					targetMsgID = part.PatchHash
					break
				}
			}
		}
		if targetMsgID != "" {
			break
		}
	}

	if targetMsgID == "" {
		m.setStatus("Nothing to revert (no snapshots recorded)")
		return m, nil
	}

	// Perform revert in a goroutine
	return m, func() tea.Msg {
		// We need the snapshot object — create a fresh one pointing at the config dirs
		snap := session.NewSnapshot(config.GetConfigDir(), config.GetProjectDir())
		if err := m.Store.Revert(m.sessionID, targetMsgID, snap); err != nil {
			return ErrorMsg{Error: fmt.Errorf("revert failed: %w", err)}
		}
		return RevertDoneMsg{}
	}
}

// RevertDoneMsg is sent when a revert completes successfully.
type RevertDoneMsg struct{}

// CompactDoneMsg is sent when a manual /compact completes successfully.
type CompactDoneMsg struct{}

func (m *Model) reinitEngine() tea.Cmd {
	// Capture fields needed (avoid closing over the whole model which is a value copy)
	providerName := m.Provider
	modelName := m.Model_
	agentName := m.Agent
	cfg := m.Config
	store := m.Store
	return func() tea.Msg {
		apiKey, keyErr := config.GetAPIKeyWithFallback(providerName, cfg)
		if keyErr != nil {
			return ErrorMsg{Error: fmt.Errorf("no API key for %s: %w", providerName, keyErr)}
		}
		prov, err := provider.CreateProvider(providerName, apiKey)
		if err != nil {
			return ErrorMsg{Error: fmt.Errorf("failed to create provider %s: %w", providerName, err)}
		}
		ag := agent.GetAgent(agentName, cfg)
		registry := tool.GetRegistry()
		// Update cfg.Provider so the engine uses the correct provider for
		// model defaults, context-window limits, and cost calculation.
		cfg.Provider = providerName
		if modelName != "" {
			cfg.Model = modelName
		}
		eng := session.NewPromptEngine(store, prov, cfg, ag, registry)
		return ProviderChangedMsg{Provider: providerName, Model: modelName, Engine: eng}
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
