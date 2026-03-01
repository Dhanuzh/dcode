package tui

// question.go — Interactive question prompt overlay, matching opencode's question.tsx
//
// The QuestionPrompt appears above the input area when the agent (or a tool)
// asks the user a structured question with a set of predefined choices (plus an
// optional "type your own answer" free-text entry).
//
// Supports:
//   - Single and multiple (multi-select) questions
//   - Multiple-choice options with optional free-text
//   - Tab navigation between questions (multi-question requests)
//   - Confirm review tab (multi-question only)
//   - Up/Down (or j/k) to move between options; Enter to select
//   - Number keys 1-9 to jump directly to an option
//   - Esc / app_exit to reject

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ─── Domain types ─────────────────────────────────────────────────────────────

// QuestionOption is one selectable item in a question.
type QuestionOption struct {
	Label       string
	Description string
}

// QuestionItem defines one question in a request.
type QuestionItem struct {
	Header   string
	Question string
	Options  []QuestionOption
	Multiple bool // allow multi-select
	Custom   bool // allow free-text entry (default true)
}

// QuestionRequest is the full request from the agent.
type QuestionRequest struct {
	ID        string
	Questions []QuestionItem
}

// ─── Messages ─────────────────────────────────────────────────────────────────

// QuestionRequestMsg is dispatched into the TUI when a question needs answering.
type QuestionRequestMsg struct {
	Req     QuestionRequest
	ReplyCh chan [][]string // answers[questionIndex][answerIndex]
}

// ─── State ────────────────────────────────────────────────────────────────────

// QuestionState holds all mutable state for the active question prompt.
type QuestionState struct {
	Active  bool
	Req     QuestionRequest
	ReplyCh chan [][]string

	Tab       int        // current question tab (or len(questions) = confirm tab)
	Selected  int        // currently highlighted option index (within current tab)
	Answers   [][]string // answers[tab][…]
	Custom    []string   // per-tab custom free-text buffer
	Editing   bool       // are we currently typing a custom answer?
	CustomBuf string     // text accumulated for the current free-text entry
}

// isSingle returns true when there is exactly one non-multiple-select question.
func (qs *QuestionState) isSingle() bool {
	return len(qs.Req.Questions) == 1 && !qs.Req.Questions[0].Multiple
}

// isConfirmTab returns true when we are on the "Confirm" summary tab.
func (qs *QuestionState) isConfirmTab() bool {
	return !qs.isSingle() && qs.Tab == len(qs.Req.Questions)
}

func (qs *QuestionState) currentQuestion() *QuestionItem {
	if qs.isConfirmTab() {
		return nil
	}
	q := &qs.Req.Questions[qs.Tab]
	return q
}

// ─── Handlers ─────────────────────────────────────────────────────────────────

// handleQuestionMsg wires a QuestionRequestMsg into the model.
func (m *Model) handleQuestionMsg(msg QuestionRequestMsg) (tea.Model, tea.Cmd) {
	nq := len(msg.Req.Questions)
	m.questionState = QuestionState{
		Active:  true,
		Req:     msg.Req,
		ReplyCh: msg.ReplyCh,
		Tab:     0,
		Answers: make([][]string, nq),
		Custom:  make([]string, nq),
	}
	return m, nil
}

// updateQuestion processes keyboard input for the active question prompt.
// Returns (handled bool, cmd).
func (m *Model) updateQuestion(key string) (bool, tea.Cmd) {
	qs := &m.questionState
	if !qs.Active {
		return false, nil
	}

	tabs := len(qs.Req.Questions)
	if !qs.isSingle() {
		tabs++ // +1 for confirm tab
	}

	// ── Custom text editing mode ─────────────────────────────────────
	if qs.Editing {
		switch key {
		case "esc":
			qs.Editing = false
			qs.CustomBuf = qs.Custom[qs.Tab]
			return true, nil
		case "enter":
			text := strings.TrimSpace(qs.CustomBuf)
			if text == "" {
				// Clear custom answer
				qs.Custom[qs.Tab] = ""
				q := qs.currentQuestion()
				if q != nil {
					qs.Answers[qs.Tab] = removeFromSlice(qs.Answers[qs.Tab], qs.Custom[qs.Tab])
				}
				qs.Editing = false
				return true, nil
			}
			qs.Custom[qs.Tab] = text
			q := qs.currentQuestion()
			if q != nil && q.Multiple {
				if !contains(qs.Answers[qs.Tab], text) {
					qs.Answers[qs.Tab] = append(qs.Answers[qs.Tab], text)
				}
			} else {
				qs.Answers[qs.Tab] = []string{text}
				if qs.isSingle() {
					return true, m.submitQuestion()
				}
				qs.Tab++
				qs.Selected = 0
			}
			qs.Editing = false
			return true, nil
		case "backspace":
			if len(qs.CustomBuf) > 0 {
				qs.CustomBuf = qs.CustomBuf[:len(qs.CustomBuf)-1]
			}
			return true, nil
		default:
			if len(key) == 1 {
				qs.CustomBuf += key
			}
			return true, nil
		}
	}

	// ── Normal mode ───────────────────────────────────────────────────
	switch key {
	case "left", "h":
		qs.Tab = (qs.Tab - 1 + tabs) % tabs
		qs.Selected = 0
		return true, nil

	case "right", "l":
		qs.Tab = (qs.Tab + 1) % tabs
		qs.Selected = 0
		return true, nil

	case "tab":
		qs.Tab = (qs.Tab + 1) % tabs
		qs.Selected = 0
		return true, nil

	case "shift+tab":
		qs.Tab = (qs.Tab - 1 + tabs) % tabs
		qs.Selected = 0
		return true, nil

	case "esc":
		return true, m.rejectQuestion()

	case "enter":
		if qs.isConfirmTab() {
			return true, m.submitQuestion()
		}
		return true, m.selectQuestionOption()

	case "up", "k":
		if !qs.isConfirmTab() {
			q := qs.currentQuestion()
			if q == nil {
				return true, nil
			}
			total := len(q.Options)
			if q.Custom {
				total++
			}
			if total > 0 {
				qs.Selected = (qs.Selected - 1 + total) % total
			}
		}
		return true, nil

	case "down", "j":
		if !qs.isConfirmTab() {
			q := qs.currentQuestion()
			if q == nil {
				return true, nil
			}
			total := len(q.Options)
			if q.Custom {
				total++
			}
			if total > 0 {
				qs.Selected = (qs.Selected + 1) % total
			}
		}
		return true, nil

	default:
		// Number keys 1-9
		if len(key) == 1 && key[0] >= '1' && key[0] <= '9' {
			idx := int(key[0] - '1')
			q := qs.currentQuestion()
			if q == nil {
				return true, nil
			}
			total := len(q.Options)
			if q.Custom {
				total++
			}
			if idx < total {
				qs.Selected = idx
				return true, m.selectQuestionOption()
			}
		}
	}
	return false, nil
}

func (m *Model) selectQuestionOption() tea.Cmd {
	qs := &m.questionState
	q := qs.currentQuestion()
	if q == nil {
		return nil
	}

	isCustom := qs.Selected == len(q.Options) && q.Custom

	if isCustom {
		qs.Editing = true
		qs.CustomBuf = qs.Custom[qs.Tab]
		return nil
	}

	if qs.Selected >= len(q.Options) {
		return nil
	}
	opt := q.Options[qs.Selected].Label

	if q.Multiple {
		if contains(qs.Answers[qs.Tab], opt) {
			qs.Answers[qs.Tab] = removeFromSlice(qs.Answers[qs.Tab], opt)
		} else {
			qs.Answers[qs.Tab] = append(qs.Answers[qs.Tab], opt)
		}
		return nil
	}

	qs.Answers[qs.Tab] = []string{opt}
	if qs.isSingle() {
		return m.submitQuestion()
	}
	qs.Tab++
	qs.Selected = 0
	return nil
}

func (m *Model) submitQuestion() tea.Cmd {
	qs := &m.questionState
	answers := make([][]string, len(qs.Req.Questions))
	for i := range answers {
		answers[i] = qs.Answers[i]
	}
	if qs.ReplyCh != nil {
		select {
		case qs.ReplyCh <- answers:
		default:
		}
	}
	m.questionState = QuestionState{}
	return nil
}

func (m *Model) rejectQuestion() tea.Cmd {
	qs := &m.questionState
	if qs.ReplyCh != nil {
		select {
		case qs.ReplyCh <- nil: // nil signals rejection
		default:
		}
	}
	m.questionState = QuestionState{}
	return nil
}

// ─── Renderer ─────────────────────────────────────────────────────────────────

// renderQuestionPrompt returns the question panel string (empty if not active).
func (m *Model) renderQuestionPrompt() string {
	qs := &m.questionState
	if !qs.Active {
		return ""
	}

	t := m.currentTheme
	w := m.viewport.Width + 2
	if w < 40 {
		w = 40
	}

	var out strings.Builder

	out.WriteString(dimStyle.Render(strings.Repeat("─", w)) + "\n")

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.Border{Left: "┃"}).
		BorderForeground(t.Primary).
		PaddingLeft(2)

	// ── Tab bar (multi-question only) ────────────────────────────────
	if !qs.isSingle() {
		var tabs []string
		for i, q := range qs.Req.Questions {
			label := q.Header
			if label == "" {
				label = fmt.Sprintf("Q%d", i+1)
			}
			if i == qs.Tab {
				tabs = append(tabs, lipgloss.NewStyle().
					Foreground(t.Background).
					Background(t.Primary).
					Padding(0, 1).
					Render(label))
			} else if len(qs.Answers[i]) > 0 {
				tabs = append(tabs, lipgloss.NewStyle().
					Foreground(t.Text).
					Padding(0, 1).
					Render(label))
			} else {
				tabs = append(tabs, lipgloss.NewStyle().
					Foreground(t.TextMuted).
					Padding(0, 1).
					Render(label))
			}
		}
		// Confirm tab
		confirmLabel := "Confirm"
		if qs.Tab == len(qs.Req.Questions) {
			tabs = append(tabs, lipgloss.NewStyle().
				Foreground(t.Background).
				Background(t.Primary).
				Padding(0, 1).
				Render(confirmLabel))
		} else {
			tabs = append(tabs, lipgloss.NewStyle().
				Foreground(t.TextMuted).
				Padding(0, 1).
				Render(confirmLabel))
		}
		out.WriteString("  " + strings.Join(tabs, " ") + "\n")
	}

	// ── Confirm review tab ────────────────────────────────────────────
	if qs.isConfirmTab() {
		var lines []string
		lines = append(lines, lipgloss.NewStyle().Foreground(t.Text).Render("Review answers:"))
		for i, q := range qs.Req.Questions {
			header := q.Header
			if header == "" {
				header = fmt.Sprintf("Q%d", i+1)
			}
			val := strings.Join(qs.Answers[i], ", ")
			if val == "" {
				val = lipgloss.NewStyle().Foreground(t.Error).Render("(not answered)")
			} else {
				val = lipgloss.NewStyle().Foreground(t.Text).Render(val)
			}
			lines = append(lines, "  "+lipgloss.NewStyle().Foreground(t.TextMuted).Render(header+": ")+val)
		}
		body := borderStyle.Render(strings.Join(lines, "\n"))
		out.WriteString(body + "\n")
	} else {
		// ── Question body ────────────────────────────────────────────
		q := qs.currentQuestion()
		if q != nil {
			suffix := ""
			if q.Multiple {
				suffix = " (select all that apply)"
			}
			questionLine := lipgloss.NewStyle().Foreground(t.Text).Render(q.Question + suffix)

			var optLines []string
			total := len(q.Options)
			for i, opt := range q.Options {
				active := i == qs.Selected
				picked := contains(qs.Answers[qs.Tab], opt.Label)

				numStyle := lipgloss.NewStyle().Foreground(t.TextMuted)
				if active {
					numStyle = numStyle.Foreground(t.Primary)
				}
				num := numStyle.Render(fmt.Sprintf("%d.", i+1))

				var labelText string
				if q.Multiple {
					check := " "
					if picked {
						check = "✓"
					}
					labelText = fmt.Sprintf("[%s] %s", check, opt.Label)
				} else {
					labelText = opt.Label
					if picked {
						labelText += " ✓"
					}
				}

				var lblStyle lipgloss.Style
				if active {
					lblStyle = lipgloss.NewStyle().Foreground(t.Primary).Background(t.Border)
				} else if picked {
					lblStyle = lipgloss.NewStyle().Foreground(t.Success)
				} else {
					lblStyle = lipgloss.NewStyle().Foreground(t.Text)
				}

				line := num + " " + lblStyle.Render(labelText)
				if opt.Description != "" {
					line += "\n   " + lipgloss.NewStyle().Foreground(t.TextMuted).Render(opt.Description)
				}
				optLines = append(optLines, line)
			}

			// Custom option
			if q.Custom {
				i := total
				active := i == qs.Selected
				customText := qs.Custom[qs.Tab]
				customPicked := customText != "" && contains(qs.Answers[qs.Tab], customText)

				numStyle := lipgloss.NewStyle().Foreground(t.TextMuted)
				if active {
					numStyle = numStyle.Foreground(t.Primary)
				}
				num := numStyle.Render(fmt.Sprintf("%d.", i+1))

				var customLabel string
				if q.Multiple {
					check := " "
					if customPicked {
						check = "✓"
					}
					customLabel = fmt.Sprintf("[%s] Type your own answer", check)
				} else {
					customLabel = "Type your own answer"
					if customPicked {
						customLabel += " ✓"
					}
				}

				var lblStyle lipgloss.Style
				if active {
					lblStyle = lipgloss.NewStyle().Foreground(t.Primary).Background(t.Border)
				} else if customPicked {
					lblStyle = lipgloss.NewStyle().Foreground(t.Success)
				} else {
					lblStyle = lipgloss.NewStyle().Foreground(t.Text)
				}

				line := num + " " + lblStyle.Render(customLabel)
				if qs.Editing {
					line += "\n   " + lipgloss.NewStyle().Foreground(t.Primary).Render(qs.CustomBuf+"█")
				} else if customText != "" {
					line += "\n   " + lipgloss.NewStyle().Foreground(t.TextMuted).Render(customText)
				}
				optLines = append(optLines, line)
			}

			body := borderStyle.Render(
				lipgloss.JoinVertical(lipgloss.Left, append([]string{questionLine}, optLines...)...),
			)
			out.WriteString(body + "\n")
		}
	}

	// ── Hint bar ─────────────────────────────────────────────────────
	var hints []string
	if !qs.isSingle() {
		hints = append(hints, lipgloss.NewStyle().Foreground(t.Text).Render("⇆")+" "+
			lipgloss.NewStyle().Foreground(t.TextMuted).Render("tab"))
	}
	if !qs.isConfirmTab() {
		hints = append(hints, lipgloss.NewStyle().Foreground(t.Text).Render("↑↓")+" "+
			lipgloss.NewStyle().Foreground(t.TextMuted).Render("select"))
	}
	enterLabel := "confirm"
	if qs.isConfirmTab() {
		enterLabel = "submit"
	} else if qs.currentQuestion() != nil && qs.currentQuestion().Multiple {
		enterLabel = "toggle"
	} else if qs.isSingle() {
		enterLabel = "submit"
	}
	hints = append(hints, lipgloss.NewStyle().Foreground(t.Text).Render("enter")+" "+
		lipgloss.NewStyle().Foreground(t.TextMuted).Render(enterLabel))
	hints = append(hints, lipgloss.NewStyle().Foreground(t.Text).Render("esc")+" "+
		lipgloss.NewStyle().Foreground(t.TextMuted).Render("dismiss"))

	out.WriteString("  " + strings.Join(hints, "   ") + "\n")

	return out.String()
}

// ─── Utility helpers ──────────────────────────────────────────────────────────

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func removeFromSlice(slice []string, s string) []string {
	out := slice[:0]
	for _, v := range slice {
		if v != s {
			out = append(out, v)
		}
	}
	return out
}
