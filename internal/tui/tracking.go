package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// TokenUsageTracker tracks token usage and costs for the current session
type TokenUsageTracker struct {
	TotalTokensIn    int
	TotalTokensOut   int
	TotalTokensCache int
	TotalCost        float64
	MaxTokens        int // Context window limit
	LastMessageTokensIn  int
	LastMessageTokensOut int
	LastMessageCost      float64
	MessageHistory   []MessageTokens
}

// MessageTokens tracks tokens for a single message
type MessageTokens struct {
	MessageID  string
	Timestamp  time.Time
	TokensIn   int
	TokensOut  int
	Cost       float64
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

	var icon string
	var statusText string
	var color lipgloss.Color

	switch m.loadingState.Type {
	case LoadingConnecting:
		icon = "🔌"
		statusText = "Connecting to " + m.Provider
		color = m.currentTheme.Info
	case LoadingGenerating:
		icon = m.spinner.View()
		statusText = "Generating response"
		color = m.currentTheme.Primary
	case LoadingToolExecution:
		icon = "⚙️"
		statusText = fmt.Sprintf("Running %s", m.loadingState.CurrentTool)
		color = m.currentTheme.Warning
	case LoadingFileOperation:
		icon = "📁"
		statusText = "Processing files"
		color = m.currentTheme.Info
	case LoadingThinking:
		icon = "💭"
		statusText = "Thinking"
		color = m.currentTheme.Secondary
	default:
		icon = m.spinner.View()
		statusText = "Processing"
		color = m.currentTheme.Text
	}

	// Add detail if available
	if m.loadingState.Detail != "" {
		statusText += " • " + m.loadingState.Detail
	}

	// Add duration
	if !m.loadingState.StartTime.IsZero() {
		duration := time.Since(m.loadingState.StartTime)
		if duration > time.Second {
			statusText += lipgloss.NewStyle().
				Foreground(m.currentTheme.TextDim).
				Render(fmt.Sprintf(" (%s)", formatDuration(duration)))
		}
	}

	statusStyle := lipgloss.NewStyle().Foreground(color)
	result := icon + " " + statusStyle.Render(statusText)

	// Add progress bar if progress is set
	if m.loadingState.Progress > 0 && m.loadingState.Progress < 1 {
		result += "\n" + m.renderProgressBar(m.loadingState.Progress)
	}

	return result
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
