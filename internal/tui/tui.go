package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/yourusername/dcode/internal/agent"
	"github.com/yourusername/dcode/internal/config"
	"github.com/yourusername/dcode/internal/provider"
	"github.com/yourusername/dcode/internal/session"
	"github.com/yourusername/dcode/internal/tool"
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

	// State
	view          View
	previousView  View
	width         int
	height        int
	sessionID     string
	messages      []session.Message
	streamingText *strings.Builder
	isStreaming   bool
	currentTool   string
	statusMsg     string
	statusExpiry  time.Time
	focusInput    bool // true = textarea focused, false = viewport focused

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

	return Model{
		viewport:      vp,
		textarea:      ta,
		spinner:       sp,
		view:          ViewChat,
		focusInput:    true,
		streamingText: &strings.Builder{},
		Agent:         agentName,
		Model_:        modelName,
		Provider:      prov,
		Store:         store,
		Engine:        engine,
		Config:        cfg,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.spinner.Tick,
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
		return m, nil

	case tea.KeyMsg:
		// Global quit
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
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
				m.view = ViewChat
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
		m.setStatus("Error: " + msg.Error.Error())
		return m, nil

	case ProviderChangedMsg:
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
	case "ctrl+k":
		m.blurTextarea()
		return m.openModelDialog()
	case "ctrl+a":
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
		{Name: "anthropic", DisplayName: "Anthropic Claude", Description: "Claude Sonnet 4, Opus 4, Haiku"},
		{Name: "openai", DisplayName: "OpenAI", Description: "GPT-4.1, GPT-4o"},
		{Name: "copilot", DisplayName: "GitHub Copilot", Description: "GPT-4 via GitHub"},
		{Name: "google", DisplayName: "Google Gemini", Description: "Gemini 2.5 Flash/Pro"},
		{Name: "groq", DisplayName: "Groq", Description: "Llama 3.3 70B (fast)"},
		{Name: "openrouter", DisplayName: "OpenRouter", Description: "Multi-provider gateway"},
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
	if sel.Connected {
		m.Provider = sel.Name
		m.Model_ = m.Config.GetDefaultModel(sel.Name)
		m.view = m.previousView
		m.focusTextarea()
		m.setStatus("Provider: " + sel.DisplayName)
		return m, m.reinitEngine()
	}
	m.setStatus("Not connected. Run: dcode login")
	m.view = m.previousView
	m.focusTextarea()
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
			return m, m.reinitEngine()
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
		m.updateViewport()
	case "tool_start":
		m.currentTool = msg.Event.ToolName
		m.updateViewport()
	case "tool_end":
		m.currentTool = ""
	case "thinking":
		// Show thinking indicator
	case "error":
		m.setStatus("Error: " + msg.Event.Content)
	case "done":
		m.isStreaming = false
		m.streamingText.Reset()
		if m.sessionID != "" {
			if sess, err := m.Store.Get(m.sessionID); err == nil {
				m.messages = sess.Messages
			}
		}
		m.updateViewport()
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
	if m.sessionID != "" {
		if sess, err := m.Store.Get(m.sessionID); err == nil {
			m.messages = sess.Messages
		}
	}
	m.updateViewport()
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
			return m, m.reinitEngine()
		}
		return m.openModelDialog()
	case "/provider":
		if len(parts) > 1 {
			m.Provider = parts[1]
			m.Model_ = m.Config.GetDefaultModel(m.Provider)
			m.setStatus("Provider: " + m.Provider)
			return m, m.reinitEngine()
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

func (m *Model) renderChat() string {
	var b strings.Builder

	// Header bar with provider / model / agent badges
	header := lipgloss.JoinHorizontal(lipgloss.Center,
		titleStyle.Render(" DCode "),
		" ",
		providerBadge.Render(m.Provider),
		" ",
		modelBadge.Render(shortModel(m.Model_)),
		" ",
		agentBadge.Render(m.Agent),
	)

	if s := m.getStatus(); s != "" {
		header += "  " + dimStyle.Render(s)
	}

	b.WriteString(header + "\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", m.width)) + "\n")

	// Messages viewport
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	// Streaming indicator
	if m.isStreaming {
		ind := m.spinner.View() + " "
		if m.currentTool != "" {
			ind += toolCallStyle.Render("⚡ " + m.currentTool)
		} else {
			ind += dimStyle.Render("Generating...")
		}
		b.WriteString(ind + "\n")
	}

	// Input area
	b.WriteString(dimStyle.Render(strings.Repeat("─", m.width)) + "\n")
	b.WriteString(m.textarea.View())
	b.WriteString("\n")

	// Footer keybindings
	// Focus indicator
	focusHint := ""
	if m.focusInput {
		focusHint = keybindStyle.Render("[INPUT]") + " "
	} else {
		focusHint = dimStyle.Render("[CHAT]") + " "
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
		{"Enter", "Send message"},
		{"Alt+Enter", "Insert newline"},
		{"Ctrl+K", "Select model"},
		{"Ctrl+J", "Cycle agent forward"},
		{"Ctrl+P", "Select provider"},
		{"Ctrl+N", "New session"},
		{"Ctrl+L", "Toggle sessions"},
		{"Ctrl+Shift+P", "Command palette"},
		{"Ctrl+,", "Settings"},
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
		marker := "  "
		if name == m.Agent {
			marker = "▸ "
		}
		b.WriteString(fmt.Sprintf("%s%s  %s\n", marker, keybindStyle.Render(fmt.Sprintf("%-12s", name)), descStyle.Render(a.Description)))
	}

	b.WriteString("\n" + dimStyle.Render("Press Esc or Q to go back"))
	return b.String()
}

// ─── Viewport content ───────────────────────────────────────────────────────────

func (m *Model) updateViewport() {
	var content strings.Builder

	for _, msg := range m.messages {
		switch msg.Role {
		case "user":
			if msg.Content != "" {
				content.WriteString(userMsgStyle.Render("You") + "\n")
				content.WriteString(msg.Content + "\n\n")
			}
			for _, part := range msg.Parts {
				if part.Type == "tool_result" {
					status := successStyle.Render("✓")
					if part.IsError {
						status = errorStyle.Render("✗")
					}
					content.WriteString(fmt.Sprintf("  %s %s\n", status, dimStyle.Render("tool result")))
					if len(part.Content) > 200 {
						content.WriteString(dimStyle.Render("    "+part.Content[:200]+"...") + "\n")
					} else if part.Content != "" {
						content.WriteString(dimStyle.Render("    "+part.Content) + "\n")
					}
				}
			}
		case "assistant":
			content.WriteString(highlightStyle.Render("DCode") + "\n")
			if msg.Content != "" {
				content.WriteString(assistantMsgStyle.Render(msg.Content) + "\n")
			}
			for _, part := range msg.Parts {
				if part.Type == "tool_use" {
					content.WriteString(toolCallStyle.Render(fmt.Sprintf("  ⚡ %s", part.ToolName)) + "\n")
				}
			}
			content.WriteString("\n")
		}
	}

	if m.isStreaming && m.streamingText.Len() > 0 {
		content.WriteString(highlightStyle.Render("DCode") + "\n")
		content.WriteString(assistantMsgStyle.Render(m.streamingText.String()))
		content.WriteString(dimStyle.Render("▊") + "\n")
	}

	m.viewport.SetContent(content.String())
	m.viewport.GotoBottom()
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
	m.isStreaming = true
	m.streamingText.Reset()

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

func clampWidth(screenW, maxW int) int {
	if screenW > maxW+10 {
		return maxW
	}
	if screenW > 30 {
		return screenW - 10
	}
	return 20
}
