package session

import (
	"strings"
)

// Compaction constants – tuned for low-credit providers like Copilot
const (
	PruneMinimum = 4000  // Minimum tokens before pruning activates
	PruneProtect = 10000 // Keep this many tokens of recent tool calls unpruned
)

// CompactionConfig holds compaction settings
type CompactionConfig struct {
	Auto  bool `json:"auto"`  // Enable automatic compaction
	Prune bool `json:"prune"` // Enable tool output pruning
}

// DefaultCompactionConfig returns the default compaction settings
func DefaultCompactionConfig() CompactionConfig {
	return CompactionConfig{
		Auto:  true,
		Prune: true,
	}
}

// IsOverflow checks if the token usage exceeds the model's usable context
func IsOverflow(inputTokens, cacheRead, outputTokens int, contextLimit, outputLimit int) bool {
	if contextLimit == 0 {
		return false
	}
	count := inputTokens + cacheRead + outputTokens

	maxOutput := outputLimit
	if maxOutput == 0 || maxOutput > OutputTokenMax {
		maxOutput = OutputTokenMax
	}

	usable := contextLimit - maxOutput
	return count > usable
}

// OutputTokenMax is the default maximum output tokens
const OutputTokenMax = 12288

// PruneProtectedTools are tools whose output should never be pruned
var PruneProtectedTools = map[string]bool{
	"skill": true,
}

// PruneToolOutputs goes backwards through messages and truncates old tool outputs
// This preserves recent tool results while compacting old ones
func PruneToolOutputs(messages []Message, pruneEnabled bool) []Message {
	if !pruneEnabled {
		return messages
	}

	var totalTokens int
	var prunedTokens int
	type pruneTarget struct {
		msgIdx  int
		partIdx int
	}
	var targets []pruneTarget

	turns := 0
	// Walk backwards through messages
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Role == "user" {
			turns++
		}
		if turns < 2 {
			continue // Protect the most recent turn
		}

		// Check for summary marker (stop pruning past summaries)
		if msg.IsSummary {
			break
		}

		for j := len(msg.Parts) - 1; j >= 0; j-- {
			part := msg.Parts[j]
			if part.Type != "tool_result" || part.IsError {
				continue
			}
			if PruneProtectedTools[part.ToolName] {
				continue
			}
			if part.IsCompacted {
				break // Already compacted, stop here
			}

			estimate := estimateTokens(part.Content)
			totalTokens += estimate

			if totalTokens > PruneProtect {
				prunedTokens += estimate
				targets = append(targets, pruneTarget{i, j})
			}
		}
	}

	// Only prune if there's enough to be worthwhile
	if prunedTokens > PruneMinimum {
		result := make([]Message, len(messages))
		copy(result, messages)

		for _, target := range targets {
			msg := result[target.msgIdx]
			parts := make([]Part, len(msg.Parts))
			copy(parts, msg.Parts)
			parts[target.partIdx].Content = "[compacted]"
			parts[target.partIdx].IsCompacted = true
			msg.Parts = parts
			result[target.msgIdx] = msg
		}
		return result
	}

	return messages
}

// estimateTokens provides a rough token estimate for text
// Approximation: 1 token per 4 characters
func estimateTokens(text string) int {
	return len(text) / 4
}

// CompactionPromptText is the default prompt sent to the compaction agent
const CompactionPromptText = `Provide a detailed prompt for continuing our conversation above. Focus on information that would be helpful for continuing the conversation, including what we did, what we're doing, which files we're working on, and what we're going to do next considering new session will not have access to our conversation.`

// BuildCompactionMessages creates the messages to send to the compaction agent
func BuildCompactionMessages(conversationMessages []Message, additionalContext []string) []Message {
	// Start with all conversation messages
	result := make([]Message, len(conversationMessages))
	copy(result, conversationMessages)

	// Add the compaction request
	promptParts := []string{CompactionPromptText}
	promptParts = append(promptParts, additionalContext...)

	result = append(result, Message{
		Role:    "user",
		Content: strings.Join(promptParts, "\n\n"),
	})

	return result
}
