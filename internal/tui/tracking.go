package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/Dhanuzh/dcode/internal/provider"
)

// TokenUsageTracker tracks token usage and costs for the current session
type TokenUsageTracker struct {
	TotalTokensIn        int
	TotalTokensOut       int
	TotalTokensCache     int
	TotalCost            float64
	MaxTokens            int // Context window limit
	LastMessageTokensIn  int
	LastMessageTokensOut int
	LastMessageCost      float64
	MessageHistory       []MessageTokens
}

// MessageTokens tracks tokens for a single message
type MessageTokens struct {
	MessageID string
	Timestamp time.Time
	TokensIn  int
	TokensOut int
	Cost      float64
}

// LoadingState represents different loading/processing states
type LoadingState struct {
	IsActive    bool
	Type        LoadingType
	Message     string
	Detail      string
	Progress    float64 // 0.0 to 1.0 for progress bar
	StartTime   time.Time
	CurrentTool string
}

// LoadingType represents the type of loading operation
type LoadingType int

const (
	LoadingNone LoadingType = iota
	LoadingConnecting
	LoadingGenerating
	LoadingToolExecution
	LoadingFileOperation
	LoadingThinking
)

// NewTokenUsageTracker creates a new token usage tracker
func NewTokenUsageTracker(maxTokens int) *TokenUsageTracker {
	return &TokenUsageTracker{
		MaxTokens:      maxTokens,
		MessageHistory: make([]MessageTokens, 0),
	}
}

// AddMessage records token usage for a message
func (t *TokenUsageTracker) AddMessage(messageID string, tokensIn, tokensOut int, cost float64) {
	t.TotalTokensIn += tokensIn
	t.TotalTokensOut += tokensOut
	t.TotalCost += cost
	t.LastMessageTokensIn = tokensIn
	t.LastMessageTokensOut = tokensOut
	t.LastMessageCost = cost

	t.MessageHistory = append(t.MessageHistory, MessageTokens{
		MessageID: messageID,
		Timestamp: time.Now(),
		TokensIn:  tokensIn,
		TokensOut: tokensOut,
		Cost:      cost,
	})

	// Keep only last 100 messages in history
	if len(t.MessageHistory) > 100 {
		t.MessageHistory = t.MessageHistory[len(t.MessageHistory)-100:]
	}
}

// GetTotalTokens returns total tokens (in + out)
func (t *TokenUsageTracker) GetTotalTokens() int {
	return t.TotalTokensIn + t.TotalTokensOut
}

// GetUsagePercentage returns the percentage of context window used
func (t *TokenUsageTracker) GetUsagePercentage() float64 {
	if t.MaxTokens == 0 {
		return 0
	}
	return float64(t.GetTotalTokens()) / float64(t.MaxTokens) * 100
}

// GetRemainingTokens returns how many tokens are left
func (t *TokenUsageTracker) GetRemainingTokens() int {
	remaining := t.MaxTokens - t.GetTotalTokens()
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Reset clears all tracking data
func (t *TokenUsageTracker) Reset() {
	t.TotalTokensIn = 0
	t.TotalTokensOut = 0
	t.TotalTokensCache = 0
	t.TotalCost = 0
	t.LastMessageTokensIn = 0
	t.LastMessageTokensOut = 0
	t.LastMessageCost = 0
	t.MessageHistory = make([]MessageTokens, 0)
}

// RenderTokenUsage renders token usage information with theme colors
func (m *Model) RenderTokenUsage() string {
	if m.tokenTracker == nil {
		return ""
	}

	total := m.tokenTracker.GetTotalTokens()
	max := m.tokenTracker.MaxTokens
	percentage := m.tokenTracker.GetUsagePercentage()

	// Choose color based on usage percentage
	var color lipgloss.Color
	var indicator string
	switch {
	case percentage >= 90:
		color = m.currentTheme.Error // Red
		indicator = "●"
	case percentage >= 70:
		color = m.currentTheme.Warning // Yellow
		indicator = "●"
	case percentage >= 50:
		color = m.currentTheme.Info // Blue
		indicator = "●"
	default:
		color = m.currentTheme.Success // Green
		indicator = "●"
	}

	// Format tokens with k/M suffix
	totalStr := formatTokens(total)
	maxStr := formatTokens(max)

	// Create enhanced display with more info
	usageStyle := lipgloss.NewStyle().Foreground(color)
	dimStyle := lipgloss.NewStyle().Foreground(m.currentTheme.TextDim)

	// Show: [indicator tokens/max (percentage%) $cost]
	display := dimStyle.Render("[") +
		usageStyle.Render(indicator) + " " +
		usageStyle.Render(fmt.Sprintf("%s/%s", totalStr, maxStr))

	// Add percentage if > 0
	if percentage > 0 {
		display += dimStyle.Render(fmt.Sprintf(" %.1f%%", percentage))
	}

	// Add cost if available
	if m.tokenTracker.TotalCost > 0 {
		display += dimStyle.Render(fmt.Sprintf(" $%.3f", m.tokenTracker.TotalCost))
	}

	display += dimStyle.Render("]")

	return display
}

// RenderDetailedTokenUsage renders detailed token usage (for status view)
func (m *Model) RenderDetailedTokenUsage() string {
	if m.tokenTracker == nil {
		return ""
	}

	var b strings.Builder
	t := m.tokenTracker

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.currentTheme.Primary)

	labelStyle := lipgloss.NewStyle().
		Foreground(m.currentTheme.TextMuted)

	valueStyle := lipgloss.NewStyle().
		Foreground(m.currentTheme.Text)

	costStyle := lipgloss.NewStyle().
		Foreground(m.currentTheme.Warning)

	b.WriteString(titleStyle.Render("📊 Token Usage") + "\n\n")

	// Total usage
	b.WriteString(labelStyle.Render("Total: "))
	b.WriteString(valueStyle.Render(fmt.Sprintf("%s / %s tokens",
		formatTokens(t.GetTotalTokens()),
		formatTokens(t.MaxTokens))))
	b.WriteString("\n")

	// Breakdown
	b.WriteString(labelStyle.Render("Input:  "))
	b.WriteString(valueStyle.Render(formatTokens(t.TotalTokensIn)))
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("Output: "))
	b.WriteString(valueStyle.Render(formatTokens(t.TotalTokensOut)))
	b.WriteString("\n\n")

	// Progress bar
	b.WriteString(m.renderTokenProgressBar())
	b.WriteString("\n\n")

	// Cost
	b.WriteString(labelStyle.Render("Total Cost: "))
	b.WriteString(costStyle.Render(fmt.Sprintf("$%.4f", t.TotalCost)))
	b.WriteString("\n\n")

	// Last message
	if t.LastMessageTokensIn > 0 || t.LastMessageTokensOut > 0 {
		b.WriteString(labelStyle.Render("Last Message:") + "\n")
		b.WriteString(labelStyle.Render("  In:  "))
		b.WriteString(valueStyle.Render(formatTokens(t.LastMessageTokensIn)))
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("  Out: "))
		b.WriteString(valueStyle.Render(formatTokens(t.LastMessageTokensOut)))
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("  Cost: "))
		b.WriteString(costStyle.Render(fmt.Sprintf("$%.4f", t.LastMessageCost)))
		b.WriteString("\n")
	}

	return b.String()
}

// renderTokenProgressBar renders a progress bar for token usage
func (m *Model) renderTokenProgressBar() string {
	if m.tokenTracker == nil {
		return ""
	}

	percentage := m.tokenTracker.GetUsagePercentage()
	barWidth := 40
	filled := int(float64(barWidth) * percentage / 100)
	if filled > barWidth {
		filled = barWidth
	}

	// Choose color based on usage
	var color lipgloss.Color
	switch {
	case percentage >= 90:
		color = m.currentTheme.Error
	case percentage >= 70:
		color = m.currentTheme.Warning
	case percentage >= 50:
		color = m.currentTheme.Info
	default:
		color = m.currentTheme.Success
	}

	filledStyle := lipgloss.NewStyle().Foreground(color)
	emptyStyle := lipgloss.NewStyle().Foreground(m.currentTheme.TextDim)

	bar := filledStyle.Render(strings.Repeat("█", filled)) +
		emptyStyle.Render(strings.Repeat("░", barWidth-filled))

	percentStr := fmt.Sprintf("%.1f%%", percentage)
	percentStyle := lipgloss.NewStyle().Foreground(m.currentTheme.TextMuted)

	return bar + " " + percentStyle.Render(percentStr)
}

// formatTokens formats token count with k/M suffix
func formatTokens(tokens int) string {
	if tokens >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(tokens)/1000000)
	}
	if tokens >= 1000 {
		return fmt.Sprintf("%.1fk", float64(tokens)/1000)
	}
	return fmt.Sprintf("%d", tokens)
}

// RenderLoadingState renders the current loading state
func (m *Model) RenderLoadingState() string {
	if !m.loadingState.IsActive {
		return ""
	}

	t := m.currentTheme
	spin := m.spinner.View()
	dim := lipgloss.NewStyle().Foreground(t.TextDim)

	switch m.loadingState.Type {
	case LoadingConnecting:
		msg := "Connecting"
		if m.loadingState.Message != "" {
			msg = m.loadingState.Message
		}
		return spin + " " + lipgloss.NewStyle().Foreground(t.Warning).Render(msg)

	case LoadingGenerating:
		return spin + " " + lipgloss.NewStyle().Foreground(t.TextMuted).Render("Generating response")

	case LoadingToolExecution:
		tool := m.loadingState.CurrentTool
		if tool == "" {
			tool = m.loadingState.Detail
		}
		line := spin + " " + lipgloss.NewStyle().Foreground(t.TextMuted).Render("Running ")
		if tool != "" {
			line += lipgloss.NewStyle().Foreground(t.Primary).Render(tool)
		}
		return line

	case LoadingThinking:
		return spin + " " + lipgloss.NewStyle().Foreground(t.Secondary).Render("Thinking...")

	case LoadingFileOperation:
		return spin + " " + lipgloss.NewStyle().Foreground(t.TextMuted).Render("Processing files")

	default:
		msg := "Processing"
		if m.loadingState.Message != "" {
			msg = m.loadingState.Message
		}
		return spin + " " + dim.Render(msg)
	}
}

// renderProgressBar renders a progress bar
func (m *Model) renderProgressBar(progress float64) string {
	barWidth := 40
	filled := int(float64(barWidth) * progress)
	if filled > barWidth {
		filled = barWidth
	}

	filledStyle := lipgloss.NewStyle().Foreground(m.currentTheme.Primary)
	emptyStyle := lipgloss.NewStyle().Foreground(m.currentTheme.TextDim)

	bar := filledStyle.Render(strings.Repeat("█", filled)) +
		emptyStyle.Render(strings.Repeat("░", barWidth-filled))

	percentStr := fmt.Sprintf("%.0f%%", progress*100)
	percentStyle := lipgloss.NewStyle().Foreground(m.currentTheme.TextMuted)

	return "  " + bar + " " + percentStyle.Render(percentStr)
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

// SetLoadingState updates the loading state
func (m *Model) SetLoadingState(loadingType LoadingType, message, detail string) {
	m.loadingState = LoadingState{
		IsActive:    true,
		Type:        loadingType,
		Message:     message,
		Detail:      detail,
		StartTime:   time.Now(),
		CurrentTool: m.currentTool,
	}
}

// SetLoadingProgress updates loading progress (0.0 to 1.0)
func (m *Model) SetLoadingProgress(progress float64) {
	m.loadingState.Progress = progress
}

// ClearLoadingState clears the loading state
func (m *Model) ClearLoadingState() {
	m.loadingState = LoadingState{IsActive: false}
}

// renderCopilotUsageView renders a detailed Copilot status and session token usage view
func (m *Model) renderCopilotUsageView(info *provider.CopilotStatusInfo) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(m.currentTheme.Primary)
	labelStyle := lipgloss.NewStyle().Foreground(m.currentTheme.TextMuted)
	valueStyle := lipgloss.NewStyle().Foreground(m.currentTheme.Text)
	successStyle := lipgloss.NewStyle().Foreground(m.currentTheme.Success)
	errorStyle := lipgloss.NewStyle().Foreground(m.currentTheme.Error)
	dimStyle := lipgloss.NewStyle().Foreground(m.currentTheme.TextDim)

	var b strings.Builder

	b.WriteString(titleStyle.Render("🐙 GitHub Copilot Status") + "\n\n")

	if info.Error != "" {
		b.WriteString(errorStyle.Render("✗ Error: "+info.Error) + "\n\n")
		b.WriteString(dimStyle.Render("Run 'dcode copilot-login' to re-authenticate.") + "\n")
		return b.String()
	}

	// Authentication info
	b.WriteString(labelStyle.Render("Account:    "))
	if info.Username != "" {
		b.WriteString(valueStyle.Render(info.Username))
	} else {
		b.WriteString(dimStyle.Render("(unknown)"))
	}
	b.WriteString("\n")

	if info.Email != "" {
		b.WriteString(labelStyle.Render("Email:      "))
		b.WriteString(valueStyle.Render(info.Email))
		b.WriteString("\n")
	}

	b.WriteString(labelStyle.Render("Plan:       "))
	if info.Plan != "" {
		b.WriteString(valueStyle.Render(info.Plan))
	} else {
		b.WriteString(dimStyle.Render("(unavailable)"))
	}
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("Status:     "))
	if info.IsActive {
		b.WriteString(successStyle.Render("✓ Active"))
	} else {
		b.WriteString(errorStyle.Render("✗ Inactive / Not subscribed"))
	}
	b.WriteString("\n\n")

	// Session token usage
	b.WriteString(titleStyle.Render("📊 Session Token Usage") + "\n\n")

	if m.tokenTracker != nil {
		t := m.tokenTracker

		b.WriteString(labelStyle.Render("Input:      "))
		b.WriteString(valueStyle.Render(formatTokens(t.TotalTokensIn)))
		b.WriteString("\n")

		b.WriteString(labelStyle.Render("Output:     "))
		b.WriteString(valueStyle.Render(formatTokens(t.TotalTokensOut)))
		b.WriteString("\n")

		b.WriteString(labelStyle.Render("Total:      "))
		b.WriteString(valueStyle.Render(formatTokens(t.GetTotalTokens())))
		if t.MaxTokens > 0 {
			b.WriteString(dimStyle.Render(fmt.Sprintf(" / %s context window", formatTokens(t.MaxTokens))))
		}
		b.WriteString("\n\n")

		// Progress bar
		b.WriteString(m.renderTokenProgressBar())
		b.WriteString("\n\n")

		// Copilot is subscription-based — no per-request cost
		b.WriteString(labelStyle.Render("Cost:       "))
		b.WriteString(dimStyle.Render("$0.00 (included in Copilot subscription)"))
		b.WriteString("\n\n")

		if len(t.MessageHistory) > 0 {
			b.WriteString(labelStyle.Render(fmt.Sprintf("Messages tracked: %d", len(t.MessageHistory))) + "\n")
		}
	} else {
		b.WriteString(dimStyle.Render("No token data available for this session.") + "\n")
	}

	b.WriteString("\n" + dimStyle.Render("Note: GitHub Copilot is subscription-based — no per-request credit limits apply."))

	return b.String()
}
