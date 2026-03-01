package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// WebSearchTool provides web search capabilities
func WebSearchTool() *ToolDef {
	return &ToolDef{
		Name:        "WebSearch",
		Description: "Search the web. Returns results with titles, URLs, and snippets.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query",
				},
				"provider": map[string]interface{}{
					"type":        "string",
					"description": "Search provider to use (duckduckgo, brave, google, bing). Defaults to duckduckgo.",
					"enum":        []string{"duckduckgo", "brave", "google", "bing"},
				},
				"max_results": map[string]interface{}{
					"type":        "number",
					"description": "Maximum number of results to return (default: 10)",
				},
			},
			"required": []string{"query"},
		},
		Execute: executeWebSearch,
	}
}

func executeWebSearch(ctx context.Context, tc *ToolContext, input map[string]interface{}) (*ToolResult, error) {
	query, ok := input["query"].(string)
	if !ok || query == "" {
		return &ToolResult{
			Output:  "query parameter is required",
			IsError: true,
		}, nil
	}

	provider := "duckduckgo"
	if p, ok := input["provider"].(string); ok {
		provider = strings.ToLower(p)
	}

	maxResults := 10
	if mr, ok := input["max_results"].(float64); ok {
		maxResults = int(mr)
	}

	// Execute search based on provider
	var results []SearchResult
	var err error

	switch provider {
	case "duckduckgo":
		results, err = searchDuckDuckGo(ctx, query, maxResults)
	case "brave":
		results, err = searchBrave(ctx, query, maxResults)
	case "google":
		results, err = searchGoogle(ctx, query, maxResults)
	case "bing":
		results, err = searchBing(ctx, query, maxResults)
	default:
		return &ToolResult{
			Output:  fmt.Sprintf("unknown search provider: %s", provider),
			IsError: true,
		}, nil
	}

	if err != nil {
		return &ToolResult{
			Output:  fmt.Sprintf("search error: %v", err),
			IsError: true,
		}, nil
	}

	// Format results
	output := formatSearchResults(results, query, provider)

	return &ToolResult{
		Output:  output,
		IsError: false,
	}, nil
}

// SearchResult represents a single search result
type SearchResult struct {
	Title   string
	URL     string
	Snippet string
}

// searchDuckDuckGo uses DuckDuckGo's HTML search (no API key required)
func searchDuckDuckGo(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
	// DuckDuckGo HTML search - parse results
	// Note: This is a simplified implementation. A production version would use a proper API
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "dcode/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("search failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse HTML results (simplified - would use a proper HTML parser in production)
	results := parseSimpleHTML(string(body), maxResults)
	return results, nil
}

// searchBrave uses Brave Search API
func searchBrave(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
	apiKey := os.Getenv("BRAVE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("BRAVE_API_KEY environment variable not set")
	}

	searchURL := fmt.Sprintf("https://api.search.brave.com/res/v1/web/search?q=%s&count=%d",
		url.QueryEscape(query), maxResults)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Subscription-Token", apiKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("brave search failed (%d): %s", resp.StatusCode, string(body))
	}

	var braveResp struct {
		Web struct {
			Results []struct {
				Title       string `json:"title"`
				URL         string `json:"url"`
				Description string `json:"description"`
			} `json:"results"`
		} `json:"web"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&braveResp); err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, len(braveResp.Web.Results))
	for _, r := range braveResp.Web.Results {
		results = append(results, SearchResult{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: r.Description,
		})
	}

	return results, nil
}

// searchGoogle uses Google Custom Search API
func searchGoogle(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
	apiKey := os.Getenv("GOOGLE_API_KEY")
	searchEngineID := os.Getenv("GOOGLE_SEARCH_ENGINE_ID")

	if apiKey == "" || searchEngineID == "" {
		return nil, fmt.Errorf("GOOGLE_API_KEY and GOOGLE_SEARCH_ENGINE_ID required")
	}

	searchURL := fmt.Sprintf("https://www.googleapis.com/customsearch/v1?key=%s&cx=%s&q=%s&num=%d",
		apiKey, searchEngineID, url.QueryEscape(query), maxResults)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("google search failed (%d): %s", resp.StatusCode, string(body))
	}

	var googleResp struct {
		Items []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&googleResp); err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, len(googleResp.Items))
	for _, item := range googleResp.Items {
		results = append(results, SearchResult{
			Title:   item.Title,
			URL:     item.Link,
			Snippet: item.Snippet,
		})
	}

	return results, nil
}

// searchBing uses Bing Search API
func searchBing(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
	apiKey := os.Getenv("BING_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("BING_API_KEY environment variable not set")
	}

	searchURL := fmt.Sprintf("https://api.bing.microsoft.com/v7.0/search?q=%s&count=%d",
		url.QueryEscape(query), maxResults)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Ocp-Apim-Subscription-Key", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("bing search failed (%d): %s", resp.StatusCode, string(body))
	}

	var bingResp struct {
		WebPages struct {
			Value []struct {
				Name    string `json:"name"`
				URL     string `json:"url"`
				Snippet string `json:"snippet"`
			} `json:"value"`
		} `json:"webPages"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&bingResp); err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, len(bingResp.WebPages.Value))
	for _, v := range bingResp.WebPages.Value {
		results = append(results, SearchResult{
			Title:   v.Name,
			URL:     v.URL,
			Snippet: v.Snippet,
		})
	}

	return results, nil
}

// parseSimpleHTML attempts to extract search results from HTML (very basic)
func parseSimpleHTML(html string, maxResults int) []SearchResult {
	// This is a very simplified parser - in production, use a proper HTML parser
	results := make([]SearchResult, 0, maxResults)

	// Look for result divs (DuckDuckGo specific patterns)
	// This is a placeholder - would need proper HTML parsing
	lines := strings.Split(html, "\n")
	for _, line := range lines {
		if len(results) >= maxResults {
			break
		}

		// Simple heuristic - look for links
		if strings.Contains(line, "http") && strings.Contains(line, "href") {
			// Extract URL and title (very basic)
			// In production, use golang.org/x/net/html
			results = append(results, SearchResult{
				Title:   "Search result (HTML parsing limited)",
				URL:     "https://example.com",
				Snippet: "Use a provider with API support for better results",
			})
		}
	}

	if len(results) == 0 {
		results = append(results, SearchResult{
			Title:   "DuckDuckGo search",
			URL:     fmt.Sprintf("https://duckduckgo.com/?q=%s", "query"),
			Snippet: "DuckDuckGo HTML parsing is limited. Consider using Brave, Google, or Bing with API keys.",
		})
	}

	return results
}

// formatSearchResults formats search results as markdown
func formatSearchResults(results []SearchResult, query, provider string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Search Results for: %s\n", query))
	sb.WriteString(fmt.Sprintf("Provider: %s | Results: %d\n\n", provider, len(results)))

	for i, result := range results {
		sb.WriteString(fmt.Sprintf("## %d. %s\n", i+1, result.Title))
		sb.WriteString(fmt.Sprintf("**URL:** %s\n", result.URL))
		if result.Snippet != "" {
			sb.WriteString(fmt.Sprintf("**Snippet:** %s\n", result.Snippet))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
