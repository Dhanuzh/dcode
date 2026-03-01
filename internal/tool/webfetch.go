package tool

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// WebFetchTool fetches web pages and converts HTML to readable text
func WebFetchTool() *ToolDef {
	return &ToolDef{
		Name:        "webfetch",
		Description: "Fetch a URL and return content as text. HTML converted to markdown. 5MB/30s limits.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "The URL to fetch",
				},
				"format": map[string]interface{}{
					"type":        "string",
					"description": "Output format: 'text' (default), 'markdown', 'html'",
					"enum":        []string{"text", "markdown", "html"},
				},
			},
			"required": []string{"url"},
		},
		Execute: func(ctx context.Context, tc *ToolContext, input map[string]interface{}) (*ToolResult, error) {
			url, _ := input["url"].(string)
			if url == "" {
				return &ToolResult{Output: "Error: url is required", IsError: true}, nil
			}

			// Validate URL
			if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
				url = "https://" + url
			}

			client := &http.Client{
				Timeout: 30 * time.Second,
			}

			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				return &ToolResult{Output: fmt.Sprintf("Error creating request: %v", err), IsError: true}, nil
			}
			req.Header.Set("User-Agent", "DCode/2.0 (AI Coding Agent)")
			req.Header.Set("Accept", "text/html,application/json,text/plain,*/*")

			resp, err := client.Do(req)
			if err != nil {
				return &ToolResult{Output: fmt.Sprintf("Error fetching URL: %v", err), IsError: true}, nil
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return &ToolResult{
					Output:  fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status),
					IsError: true,
				}, nil
			}

			// Read with size limit (5MB)
			limitReader := io.LimitReader(resp.Body, 5*1024*1024)
			body, err := io.ReadAll(limitReader)
			if err != nil {
				return &ToolResult{Output: fmt.Sprintf("Error reading response: %v", err), IsError: true}, nil
			}

			content := string(body)
			contentType := resp.Header.Get("Content-Type")

			format := "text"
			if v, ok := input["format"].(string); ok && v != "" {
				format = v
			}

			// Convert HTML to readable text
			if strings.Contains(contentType, "text/html") && format != "html" {
				content = htmlToText(content)
			}

			// Truncate if too large
			if len(content) > 100*1024 {
				content = content[:100*1024] + "\n\n... (content truncated at 100KB)"
			}

			header := fmt.Sprintf("URL: %s\nContent-Type: %s\nSize: %d bytes\n\n", url, contentType, len(body))
			return &ToolResult{Output: header + content}, nil
		},
	}
}

// htmlToText converts HTML to readable plain text
func htmlToText(html string) string {
	// Remove script and style tags
	scriptRe := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	html = scriptRe.ReplaceAllString(html, "")
	styleRe := regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	html = styleRe.ReplaceAllString(html, "")

	// Convert headings
	for i := 6; i >= 1; i-- {
		re := regexp.MustCompile(fmt.Sprintf(`(?is)<h%d[^>]*>(.*?)</h%d>`, i, i))
		prefix := strings.Repeat("#", i)
		html = re.ReplaceAllString(html, "\n"+prefix+" $1\n")
	}

	// Convert paragraphs and divs to newlines
	pRe := regexp.MustCompile(`(?is)<(?:p|div)[^>]*>`)
	html = pRe.ReplaceAllString(html, "\n")
	pCloseRe := regexp.MustCompile(`(?is)</(?:p|div)>`)
	html = pCloseRe.ReplaceAllString(html, "\n")

	// Convert br tags
	brRe := regexp.MustCompile(`(?is)<br\s*/?>`)
	html = brRe.ReplaceAllString(html, "\n")

	// Convert line items
	liRe := regexp.MustCompile(`(?is)<li[^>]*>`)
	html = liRe.ReplaceAllString(html, "\n- ")

	// Convert links
	linkRe := regexp.MustCompile(`(?is)<a[^>]*href="([^"]*)"[^>]*>(.*?)</a>`)
	html = linkRe.ReplaceAllString(html, "$2 ($1)")

	// Convert bold/strong
	boldRe := regexp.MustCompile(`(?is)<(?:b|strong)[^>]*>(.*?)</(?:b|strong)>`)
	html = boldRe.ReplaceAllString(html, "**$1**")

	// Convert italic/em
	italicRe := regexp.MustCompile(`(?is)<(?:i|em)[^>]*>(.*?)</(?:i|em)>`)
	html = italicRe.ReplaceAllString(html, "*$1*")

	// Convert code
	codeRe := regexp.MustCompile(`(?is)<code[^>]*>(.*?)</code>`)
	html = codeRe.ReplaceAllString(html, "`$1`")

	// Convert pre blocks
	preRe := regexp.MustCompile(`(?is)<pre[^>]*>(.*?)</pre>`)
	html = preRe.ReplaceAllString(html, "\n```\n$1\n```\n")

	// Remove remaining HTML tags
	tagRe := regexp.MustCompile(`<[^>]+>`)
	html = tagRe.ReplaceAllString(html, "")

	// Decode common HTML entities
	html = strings.ReplaceAll(html, "&amp;", "&")
	html = strings.ReplaceAll(html, "&lt;", "<")
	html = strings.ReplaceAll(html, "&gt;", ">")
	html = strings.ReplaceAll(html, "&quot;", "\"")
	html = strings.ReplaceAll(html, "&#39;", "'")
	html = strings.ReplaceAll(html, "&nbsp;", " ")

	// Clean up whitespace
	multiNewline := regexp.MustCompile(`\n{3,}`)
	html = multiNewline.ReplaceAllString(html, "\n\n")
	html = strings.TrimSpace(html)

	return html
}
