package srv

import (
	"bytes"
	"context"
	"net/http/cookiejar"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-shiori/go-readability"
)


type ArticleIngestRequest struct {
	URL  string `json:"url"`
	Text string `json:"text"`
}

type ArticleIngestResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	Title    string `json:"title,omitempty"`
	Appended bool   `json:"appended"`
}

// HandleIngest processes POST /ingest requests
func (s *Server) HandleIngest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check rate limit
	if !s.limiter.Allow() {
		w.WriteHeader(http.StatusTooManyRequests)
		slog.Warn("rate limit exceeded", "ip", r.RemoteAddr)
		json.NewEncoder(w).Encode(ArticleIngestResponse{
			Status:  "error",
			Message: "rate limit exceeded",
		})
		return
	}

	// Check API token
	token := r.URL.Query().Get("token")
	if s.apiToken != "" && token != s.apiToken {
		w.WriteHeader(http.StatusUnauthorized)
		slog.Warn("unauthorized access attempt", "ip", r.RemoteAddr)
		json.NewEncoder(w).Encode(ArticleIngestResponse{
			Status:  "error",
			Message: "unauthorized",
		})
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ArticleIngestResponse{
			Status:  "error",
			Message: "POST method required",
		})
		return
	}

	// Read raw body for debugging
	bodyBytes, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	
	slog.Info("raw request body", "content_type", r.Header.Get("Content-Type"), "body", string(bodyBytes[:min(len(bodyBytes), 200)]))
	
	var req ArticleIngestRequest
	contentType := r.Header.Get("Content-Type")
	
	// Handle both JSON and URL-encoded form data
	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		// Parse form data
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			slog.Warn("failed to parse form data", "error", err)
			json.NewEncoder(w).Encode(ArticleIngestResponse{
				Status:  "error",
				Message: "invalid form data",
			})
			return
		}
		req.URL = strings.TrimSpace(r.FormValue("url"))
		req.Text = strings.TrimSpace(r.FormValue("text"))
	} else {
		// Parse JSON
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			slog.Warn("failed to decode JSON", "error", err)
			json.NewEncoder(w).Encode(ArticleIngestResponse{
				Status:  "error",
				Message: "invalid JSON",
			})
			return
		}
	}

	// Log the incoming request for debugging
	slog.Info("received ingest request",
		"url_field", req.URL,
		"text_field_preview", func() string {
			if len(req.Text) > 100 {
				return req.Text[:100] + "..."
			}
			return req.Text
		}())

	// Extract URL from either url or text field
	articleURL := strings.TrimSpace(req.URL)
	if articleURL == "" && req.Text != "" {
		var err error
		articleURL, err = extractURLFromText(req.Text)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			slog.Warn("failed to extract URL from text", "error", err)
			json.NewEncoder(w).Encode(ArticleIngestResponse{
				Status:  "error",
				Message: fmt.Sprintf("failed to extract URL: %v", err),
			})
			return
		}
	}

	if articleURL == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ArticleIngestResponse{
			Status:  "error",
			Message: "url or text field required",
		})
		return
	}

	// Validate URL
	if err := validateURL(articleURL); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		slog.Warn("invalid URL", "url", articleURL, "error", err)
		json.NewEncoder(w).Encode(ArticleIngestResponse{
			Status:  "error",
			Message: fmt.Sprintf("invalid URL: %v", err),
		})
		return
	}

	// Fetch the article
	slog.Info("ingest request", "url", articleURL)
	article, err := fetchArticle(articleURL)
	if err != nil {
		slog.Warn("failed to fetch article", "url", articleURL, "error", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ArticleIngestResponse{
			Status:  "error",
			Message: fmt.Sprintf("failed to fetch article: %v", err),
		})
		return
	}

	// Synthesize via LLM gateway
	synthesis, connections, err := synthesizeAndCheckConnections(article)
	if err != nil {
		slog.Warn("synthesis failed", "title", article.Title, "error", err)
		synthesis = fmt.Sprintf("(Fetch only, LLM unavailable) %s", article.Title)
	}

	// Append to daily log
	if err := appendToDaily(article.Title, synthesis, connections); err != nil {
		slog.Warn("failed to append to daily", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ArticleIngestResponse{
			Status:  "error",
			Message: fmt.Sprintf("failed to append to log: %v", err),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ArticleIngestResponse{
		Status:   "success",
		Message:  "article ingested and appended to daily log",
		Title:    article.Title,
		Appended: true,
	})
}

// validateURL checks if a URL is safe to fetch
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// extractURLFromText finds the first http(s) URL in text
func extractURLFromText(text string) (string, error) {
	re := regexp.MustCompile(`https?://[^\s\)<>]+`)
	matches := re.FindStringSubmatch(text)
	if len(matches) == 0 {
		return "", fmt.Errorf("no URL found in text")
	}
	// Clean up trailing punctuation
	url := strings.TrimRight(matches[0], ".,;:!?)\"'")
	return url, nil
}

func validateURL(articleURL string) error {
	parsedURL, err := url.Parse(articleURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	// Reject non-HTTP(S)
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("only HTTP(S) URLs allowed, got %q", parsedURL.Scheme)
	}

	hostname := parsedURL.Hostname()
	if hostname == "" {
		return fmt.Errorf("URL has no hostname")
	}

	// Reject localhost and private IP ranges (SSRF protection)
	if hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1" {
		return fmt.Errorf("localhost not allowed")
	}

	// Reject private IP ranges
	if strings.HasPrefix(hostname, "127.") ||
		strings.HasPrefix(hostname, "10.") ||
		strings.HasPrefix(hostname, "172.") ||
		strings.HasPrefix(hostname, "192.168.") ||
		strings.HasPrefix(hostname, "169.254.") { // Link-local
		return fmt.Errorf("private IP ranges not allowed")
	}

	// Reject file:// and data: schemes (defense in depth)
	if parsedURL.Scheme == "file" || parsedURL.Scheme == "data" {
		return fmt.Errorf("scheme %q not allowed", parsedURL.Scheme)
	}

	return nil
}

// expandShortenedURL follows redirects to get the final URL
func expandShortenedURL(shortURL string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow up to 10 redirects
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, shortURL, nil)
	if err != nil {
		return shortURL, nil
	}

	// Spoof Chrome browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return shortURL, nil
	}
	defer resp.Body.Close()

	// Return the final URL after all redirects
	return resp.Request.URL.String(), nil
}

func fetchArticle(articleURL string) (*readability.Article, error) {
	// Try markdown variant first (Cloudflare-friendly)
	if !strings.HasSuffix(articleURL, ".md") && !strings.Contains(articleURL, "share.google") {
		markdownURL := articleURL + ".md"
		resp, err := http.Head(markdownURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			slog.Info("markdown variant available", "url", markdownURL)
			articleURL = markdownURL
		} else if resp != nil {
			resp.Body.Close()
		}
	}

	// Expand shortened URLs (e.g., share.google)
	if strings.Contains(articleURL, "share.google/") {
		expanded, err := expandShortenedURL(articleURL)
		if expanded != "" && expanded != articleURL {
			slog.Info("expanded shortened URL", "short", articleURL, "expanded", expanded)
			articleURL = expanded
		} else if err != nil {
			slog.Warn("failed to expand URL", "url", articleURL, "error", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Create a client with cookie jar for realistic browser behavior
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:     jar,
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, articleURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Spoof realistic Chrome browser to bypass anti-bot detection
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7,text/markdown;q=0.9")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Sec-Ch-Ua", `"Not_A Brand";v="8", "Chromium";v="120", "Microsoft Edge";v="120"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("DNT", "1")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Referer", "https://www.google.com/")
	req.Header.Set("Origin", "https://www.google.com")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// Limit response body size to 10MB to prevent resource exhaustion
	limitedBody := io.LimitReader(resp.Body, 10*1024*1024)
	body, err := io.ReadAll(limitedBody)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	// Parse URL for readability
	parsedURL, err := url.Parse(articleURL)
	if err != nil {
		return nil, fmt.Errorf("parse URL: %w", err)
	}

	article, err := readability.FromReader(bytes.NewReader(body), parsedURL)
	if err != nil {
		return nil, fmt.Errorf("parse article: %w", err)
	}

	return &article, nil
}

func synthesizeAndCheckConnections(article *readability.Article) (string, string, error) {
	title := article.Title
	content := article.TextContent

	// Truncate if too long
	if len(content) > 4000 {
		content = content[:4000] + "...[truncated]"
	}

	// Create synthesis prompt
	prompt := fmt.Sprintf("Synthesize this article in 2-3 sentences, focusing on key insights:\n\nTitle: %s\n\nContent:\n%s", title, content)

	synthesis, err := callLLMGateway(prompt)
	if err != nil {
		return "", "", err
	}

	// Check against LONGTERM.md for active research threads
	connections := checkResearchConnections(title, article.TextContent)

	return synthesis, connections, nil
}

func callLLMGateway(prompt string) (string, error) {
	type Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	type Request struct {
		Model     string    `json:"model"`
		MaxTokens int       `json:"max_tokens"`
		System    string    `json:"system"`
		Messages  []Message `json:"messages"`
	}

	type ContentBlock struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}

	type LLMResponse struct {
		Content []ContentBlock `json:"content"`
		Error   *struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}

	reqBody := Request{
		Model:     "claude-sonnet-4-6",
		MaxTokens: 500,
		System:    "You are a research synthesis assistant. Provide clear, concise summaries.",
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	// Use exe.dev LLM gateway (no API key required, included in subscription)
	httpReq, err := http.NewRequest("POST", "http://169.254.169.254/gateway/llm/anthropic/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return "", err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("gateway request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var llmResp LLMResponse
	if err := json.Unmarshal(body, &llmResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w (body: %s)", err, string(body))
	}

	if llmResp.Error != nil {
		return "", fmt.Errorf("LLM gateway: %s (%s)", llmResp.Error.Message, llmResp.Error.Type)
	}

	if len(llmResp.Content) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	return llmResp.Content[0].Text, nil
}

func checkResearchConnections(title, content string) string {
	// Read LONGTERM.md and scan for relevant keywords
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	longtermPath := filepath.Join(home, ".agent", "memory", "LONGTERM.md")
	longtermBytes, err := os.ReadFile(longtermPath)
	if err != nil {
		return "" // File not found, no connections to report
	}

	_ = string(longtermBytes) // Variable for reference; we do simple keyword matching

	// Simple keyword matching for research threads
	keywords := map[string]string{
		"agent governance":         "Agent Governance & Failure Modes Research",
		"virtue ethics":            "Virtue Ethics as Infrastructure Design",
		"attractor basin":          "Attractor basins as cognition metaphor",
		"memory tier":              "Memory Research — Comparative Tiering",
		"compaction":               "Compaction Resilience as Hard Problem",
		"verification":             "Verification-as-Constitution",
		"classification problem":   "Astral's Classification Problem",
		"multi-agent":              "Governance Patterns from Multi-Agent Research",
		"anomaly detection":        "Argos Anomaly Detection",
		"operator investment":      "Operator investment multiplicative dynamics",
		"constraint":               "Governance as Repeated Deliberation",
	}

	var matchedThreads []string
	contentLower := strings.ToLower(content)
	titleLower := strings.ToLower(title)

	for keyword, threadName := range keywords {
		if strings.Contains(contentLower, keyword) || strings.Contains(titleLower, keyword) {
			matchedThreads = append(matchedThreads, threadName)
		}
	}

	if len(matchedThreads) == 0 {
		return ""
	}

	// Return first matched thread (one sentence max for daily log)
	return "Relates to: " + matchedThreads[0]
}

func appendToDaily(title, synthesis, connections string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}

	dailyPath := filepath.Join(home, ".agent", "memory", "daily", fmt.Sprintf("%s.md", time.Now().Format("2006-01-02")))

	// Create parent directory if needed
	if err := os.MkdirAll(filepath.Dir(dailyPath), 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Read existing content if file exists
	var existingContent []byte
	if info, err := os.Stat(dailyPath); err == nil && info.Size() > 0 {
		var readErr error
		existingContent, readErr = os.ReadFile(dailyPath)
		if readErr != nil {
			return fmt.Errorf("read existing log: %w", readErr)
		}
	}

	// Format entry as ## Shaun shared — HH:MM
	now := time.Now()
	timeStr := now.Format("15:04")
	entry := fmt.Sprintf("\n## Shaun shared — %s\n\n%s", timeStr, synthesis)

	if connections != "" {
		entry += fmt.Sprintf("\n\n*%s*", connections)
	}

	// Append to file
	newContent := string(existingContent) + entry + "\n"
	if err := os.WriteFile(dailyPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("write log: %w", err)
	}

	slog.Info("appended to daily log", "path", dailyPath, "title", title)
	return nil
}
