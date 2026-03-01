package tool

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

// Replacer is a function that takes file content and a search string,
// and returns candidate strings that might match in the content.
// Each candidate is tried via strings.Index(content, candidate).
// Replacers are tried in priority order; the first unique match wins.
type Replacer func(content, find string) []string

// levenshtein computes the Levenshtein edit distance between two strings.
func levenshtein(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Use two-row DP for space efficiency
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)

	for j := 0; j <= len(b); j++ {
		prev[j] = j
	}

	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min3(
				prev[j]+1,      // deletion
				curr[j-1]+1,    // insertion
				prev[j-1]+cost, // substitution
			)
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// Strategy 1: SimpleReplacer - exact match (identity).
func simpleReplacer(_ string, find string) []string {
	return []string{find}
}

// Strategy 2: LineTrimmedReplacer - matches lines after trimming whitespace.
// When the LLM gets code right but uses wrong indentation.
func lineTrimmedReplacer(content string, find string) []string {
	contentLines := strings.Split(content, "\n")
	searchLines := strings.Split(find, "\n")
	if len(searchLines) == 0 {
		return nil
	}

	var results []string

	for i := 0; i <= len(contentLines)-len(searchLines); i++ {
		match := true
		for j := 0; j < len(searchLines); j++ {
			if strings.TrimSpace(contentLines[i+j]) != strings.TrimSpace(searchLines[j]) {
				match = false
				break
			}
		}
		if match {
			// Yield the original content preserving indentation
			matched := make([]string, len(searchLines))
			for j := 0; j < len(searchLines); j++ {
				matched[j] = contentLines[i+j]
			}
			results = append(results, strings.Join(matched, "\n"))
		}
	}
	return results
}

// Strategy 3: BlockAnchorReplacer - uses first/last lines as anchors
// with Levenshtein-based fuzzy matching for middle lines.
func blockAnchorReplacer(content string, find string) []string {
	const singleCandidateThreshold = 0.0
	const multipleCandidateThreshold = 0.3

	searchLines := strings.Split(find, "\n")
	if len(searchLines) < 3 {
		return nil
	}

	contentLines := strings.Split(content, "\n")
	firstLine := strings.TrimSpace(searchLines[0])
	lastLine := strings.TrimSpace(searchLines[len(searchLines)-1])

	type candidate struct {
		text       string
		similarity float64
	}

	var candidates []candidate

	for i := 0; i < len(contentLines); i++ {
		if strings.TrimSpace(contentLines[i]) != firstLine {
			continue
		}
		// Look for the last line after this position
		for endIdx := i + len(searchLines) - 1; endIdx < len(contentLines) && endIdx < i+len(searchLines)*2; endIdx++ {
			if strings.TrimSpace(contentLines[endIdx]) != lastLine {
				continue
			}
			// Found potential block from i to endIdx
			blockLines := contentLines[i : endIdx+1]
			block := strings.Join(blockLines, "\n")

			// Compute similarity for middle lines
			middleSearch := searchLines[1 : len(searchLines)-1]
			middleBlock := blockLines[1 : len(blockLines)-1]

			sim := computeBlockSimilarity(middleSearch, middleBlock)
			candidates = append(candidates, candidate{text: block, similarity: sim})
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	// Pick the best candidate
	threshold := singleCandidateThreshold
	if len(candidates) > 1 {
		threshold = multipleCandidateThreshold
	}

	// Sort by similarity descending, return those above threshold
	var results []string
	bestSim := -1.0
	bestIdx := 0
	for i, c := range candidates {
		if c.similarity > bestSim {
			bestSim = c.similarity
			bestIdx = i
		}
	}

	if bestSim >= threshold {
		results = append(results, candidates[bestIdx].text)
	}

	return results
}

// computeBlockSimilarity computes line-by-line similarity between two blocks
// using Levenshtein distance. Returns a score between 0 and 1.
func computeBlockSimilarity(searchLines, blockLines []string) float64 {
	if len(searchLines) == 0 && len(blockLines) == 0 {
		return 1.0
	}
	if len(searchLines) == 0 || len(blockLines) == 0 {
		return 0.0
	}

	maxLines := len(searchLines)
	if len(blockLines) > maxLines {
		maxLines = len(blockLines)
	}

	totalSim := 0.0
	for i := 0; i < maxLines; i++ {
		var s, b string
		if i < len(searchLines) {
			s = strings.TrimSpace(searchLines[i])
		}
		if i < len(blockLines) {
			b = strings.TrimSpace(blockLines[i])
		}

		if s == "" && b == "" {
			totalSim += 1.0
			continue
		}

		maxLen := len(s)
		if len(b) > maxLen {
			maxLen = len(b)
		}
		if maxLen == 0 {
			totalSim += 1.0
			continue
		}

		dist := levenshtein(s, b)
		totalSim += 1.0 - float64(dist)/float64(maxLen)
	}

	return totalSim / float64(maxLines)
}

// Strategy 4: WhitespaceNormalizedReplacer - normalizes all whitespace to single spaces.
func whitespaceNormalizedReplacer(content string, find string) []string {
	normalizedFind := normalizeWhitespace(find)
	if normalizedFind == "" {
		return nil
	}

	contentLines := strings.Split(content, "\n")
	searchLines := strings.Split(find, "\n")

	// Single-line case: search each content line with normalized whitespace
	if len(searchLines) == 1 {
		var results []string
		// Build regex from words
		words := strings.Fields(normalizedFind)
		if len(words) == 0 {
			return nil
		}
		escaped := make([]string, len(words))
		for i, w := range words {
			escaped[i] = regexp.QuoteMeta(w)
		}
		pattern := strings.Join(escaped, `\s+`)
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil
		}
		for _, line := range contentLines {
			if m := re.FindString(line); m != "" {
				results = append(results, m)
			}
		}
		return results
	}

	// Multi-line case: normalize both and find blocks
	normalizedContent := normalizeWhitespace(content)
	idx := strings.Index(normalizedContent, normalizedFind)
	if idx == -1 {
		return nil
	}

	// Map back to original content
	// Find the block in original content whose normalized form matches
	var results []string
	for i := 0; i <= len(contentLines)-len(searchLines); i++ {
		block := strings.Join(contentLines[i:i+len(searchLines)], "\n")
		if normalizeWhitespace(block) == normalizedFind {
			results = append(results, block)
		}
	}

	return results
}

func normalizeWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// Strategy 5: IndentationFlexibleReplacer - strips minimum common indentation
// then compares. Handles wrong nesting level with correct relative indentation.
func indentationFlexibleReplacer(content string, find string) []string {
	normalizedFind := removeIndentation(find)
	if normalizedFind == "" {
		return nil
	}

	contentLines := strings.Split(content, "\n")
	searchLines := strings.Split(normalizedFind, "\n")

	var results []string
	for i := 0; i <= len(contentLines)-len(searchLines); i++ {
		block := strings.Join(contentLines[i:i+len(searchLines)], "\n")
		if removeIndentation(block) == normalizedFind {
			results = append(results, block)
		}
	}
	return results
}

func removeIndentation(text string) string {
	lines := strings.Split(text, "\n")
	// Find minimum indentation across non-empty lines
	minIndent := math.MaxInt32
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed == "" {
			continue // skip empty lines
		}
		indent := len(line) - len(trimmed)
		if indent < minIndent {
			minIndent = indent
		}
	}
	if minIndent == math.MaxInt32 || minIndent == 0 {
		return text
	}

	// Strip minIndent from each line
	result := make([]string, len(lines))
	for i, line := range lines {
		if len(line) >= minIndent {
			result[i] = line[minIndent:]
		} else {
			result[i] = strings.TrimLeft(line, " \t")
		}
	}
	return strings.Join(result, "\n")
}

// Strategy 6: EscapeNormalizedReplacer - unescapes common escape sequences.
// Handles cases where the LLM double-escaped characters.
func escapeNormalizedReplacer(content string, find string) []string {
	unescaped := unescapeString(find)
	if unescaped == find {
		return nil // No change after unescaping, skip
	}

	var results []string
	if strings.Contains(content, unescaped) {
		results = append(results, unescaped)
	}

	// Also try unescaping both sides
	unescapedContent := unescapeString(content)
	if unescapedContent != content && strings.Contains(unescapedContent, find) {
		// Find the original text segment that corresponds
		// This is a simplified version - we check if find exists in unescaped content
		results = append(results, find)
	}

	return results
}

func unescapeString(s string) string {
	replacements := map[string]string{
		`\n`:  "\n",
		`\t`:  "\t",
		`\r`:  "\r",
		`\'`:  "'",
		`\"`:  "\"",
		"\\`": "`",
		`\\`:  "\\",
		`\$`:  "$",
	}
	result := s
	for old, new_ := range replacements {
		result = strings.ReplaceAll(result, old, new_)
	}
	return result
}

// Strategy 7: TrimmedBoundaryReplacer - trims the entire search block.
// Handles extra leading/trailing whitespace on the whole block.
func trimmedBoundaryReplacer(content string, find string) []string {
	trimmed := strings.TrimSpace(find)
	if trimmed == find {
		return nil // No change after trimming
	}
	if trimmed == "" {
		return nil
	}

	var results []string
	if strings.Contains(content, trimmed) {
		results = append(results, trimmed)
	}

	// Also try line-by-line trimmed block matching
	contentLines := strings.Split(content, "\n")
	searchLines := strings.Split(trimmed, "\n")
	for i := 0; i <= len(contentLines)-len(searchLines); i++ {
		block := strings.Join(contentLines[i:i+len(searchLines)], "\n")
		if strings.TrimSpace(block) == trimmed {
			results = append(results, block)
		}
	}
	return results
}

// Strategy 8: ContextAwareReplacer - uses first/last line anchors with
// same line count constraint and 50% exact middle line match requirement.
func contextAwareReplacer(content string, find string) []string {
	searchLines := strings.Split(find, "\n")
	if len(searchLines) < 3 {
		return nil
	}

	contentLines := strings.Split(content, "\n")
	firstLine := strings.TrimSpace(searchLines[0])
	lastLine := strings.TrimSpace(searchLines[len(searchLines)-1])

	var results []string

	for i := 0; i <= len(contentLines)-len(searchLines); i++ {
		if strings.TrimSpace(contentLines[i]) != firstLine {
			continue
		}
		endIdx := i + len(searchLines) - 1
		if endIdx >= len(contentLines) {
			continue
		}
		if strings.TrimSpace(contentLines[endIdx]) != lastLine {
			continue
		}

		// Same line count — check middle lines
		middleSearch := searchLines[1 : len(searchLines)-1]
		middleContent := contentLines[i+1 : endIdx]

		totalNonEmpty := 0
		matchingLines := 0
		for j := 0; j < len(middleSearch); j++ {
			if strings.TrimSpace(middleSearch[j]) == "" {
				continue
			}
			totalNonEmpty++
			if j < len(middleContent) && strings.TrimSpace(middleContent[j]) == strings.TrimSpace(middleSearch[j]) {
				matchingLines++
			}
		}

		if totalNonEmpty == 0 || float64(matchingLines)/float64(totalNonEmpty) >= 0.5 {
			block := strings.Join(contentLines[i:endIdx+1], "\n")
			results = append(results, block)
			break // Only match first occurrence
		}
	}
	return results
}

// Strategy 9: MultiOccurrenceReplacer - finds all exact occurrences.
// Last resort for replaceAll scenarios.
func multiOccurrenceReplacer(_ string, find string) []string {
	// This just returns the find string — the orchestrator uses it
	// to handle multiple matches with replaceAll.
	return []string{find}
}

// allReplacers returns all 9 strategies in priority order.
func allReplacers() []Replacer {
	return []Replacer{
		simpleReplacer,
		lineTrimmedReplacer,
		blockAnchorReplacer,
		whitespaceNormalizedReplacer,
		indentationFlexibleReplacer,
		escapeNormalizedReplacer,
		trimmedBoundaryReplacer,
		contextAwareReplacer,
		multiOccurrenceReplacer,
	}
}

// FuzzyReplace attempts to find and replace oldString in content using
// progressively relaxed matching strategies. If replaceAll is true,
// it replaces all occurrences. Returns the new content and an error
// if no match is found.
func FuzzyReplace(content, oldString, newString string, replaceAll bool) (string, error) {
	replacers := allReplacers()
	foundMultiple := false

	for _, replacer := range replacers {
		candidates := replacer(content, oldString)
		for _, candidate := range candidates {
			idx := strings.Index(content, candidate)
			if idx == -1 {
				continue
			}

			if replaceAll {
				return strings.ReplaceAll(content, candidate, newString), nil
			}

			// Check uniqueness — the candidate must match exactly once
			lastIdx := strings.LastIndex(content, candidate)
			if idx != lastIdx {
				foundMultiple = true
				continue
			}

			return strings.Replace(content, candidate, newString, 1), nil
		}
	}

	if foundMultiple {
		return "", fmt.Errorf("Found multiple matches for oldString. Provide more surrounding lines in oldString to identify the correct match.")
	}
	return "", fmt.Errorf("oldString not found in content")
}
