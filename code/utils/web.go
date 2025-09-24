package utils

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// FetchURLAsPlainText fetches the given URL and returns readable text.
// It leverages Jina Reader (https://r.jina.ai) to extract clean content without HTML.
// The input can be with or without scheme; we'll normalize it.
func FetchURLAsPlainText(rawURL string) (string, error) {
	cleaned := strings.TrimSpace(rawURL)
	if cleaned == "" {
		return "", errors.New("empty url")
	}

	// Normalize into Jina Reader endpoint
	// Jina Reader format: https://r.jina.ai/http://example.com or https://r.jina.ai/https://example.com
	var readerURL string
	if strings.HasPrefix(cleaned, "http://") || strings.HasPrefix(cleaned, "https://") {
		readerURL = "https://r.jina.ai/" + cleaned
	} else {
		// default to https
		readerURL = "https://r.jina.ai/https://" + cleaned
	}

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", readerURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", errors.New("failed to fetch url: " + resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// SearchResult represents a single search result
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet,omitempty"`
}

// WebSearch performs a web search using DuckDuckGo's HTML endpoint via Jina Reader aggregator API.
// We use https://r.jina.ai/http://r.jina.ai/http://duckduckgo.com/html/?q=... pattern is not valid; instead we'll call a lightweight meta-search API.
// Here we leverage Brave Search API compatible relay by Jina: https://r.jina.ai/http://r.jina.ai/http://r.jina.ai/... is not reliable in China, so fallback to DuckDuckGo's lite HTML and extract links is complex.
// To keep it simple and dependency-free, we use DuckDuckGo's lite HTML and let r.jina.ai convert to text, then pick top lines containing http.
func WebSearch(query string, topK int) ([]SearchResult, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, errors.New("empty query")
	}
	if topK <= 0 {
		topK = 3
	}
	// Use DuckDuckGo HTML with no JS
	// ddg lite: https://duckduckgo.com/html/?q=
	duckURL := "https://duckduckgo.com/html/?q=" + url.QueryEscape(q)
	txt, err := FetchURLAsPlainText(duckURL)
	if err != nil {
		return nil, err
	}
	// crude extraction: find lines with http(s) and a title-like preceding line
	lines := strings.Split(txt, "\n")
	var results []SearchResult
	var lastTitle string
	for _, line := range lines {
		l := strings.TrimSpace(line)
		if l == "" {
			continue
		}
		if strings.HasPrefix(l, "http://") || strings.HasPrefix(l, "https://") {
			if !strings.Contains(l, "duckduckgo.com") {
				results = append(results, SearchResult{Title: lastTitle, URL: l})
				if len(results) >= topK {
					break
				}
			}
			lastTitle = ""
			continue
		}
		// update possible title
		if len(l) > 0 && len(l) <= 200 {
			lastTitle = l
		}
	}
	if len(results) == 0 {
		return nil, errors.New("no results")
	}
	return results, nil
}

// BuildSearchContext downloads topK results and returns a concatenated context string.
func BuildSearchContext(query string, topK int) (string, error) {
	results, err := WebSearch(query, topK)
	if err != nil {
		return "", err
	}
	type item struct {
		Title   string `json:"title"`
		URL     string `json:"url"`
		Content string `json:"content"`
	}
	var items []item
	for _, r := range results {
		content, err := FetchURLAsPlainText(r.URL)
		if err != nil {
			continue
		}
		items = append(items, item{Title: r.Title, URL: r.URL, Content: content})
	}
	if len(items) == 0 {
		return "", errors.New("no accessible results")
	}
	b, _ := json.Marshal(items)
	return string(b), nil
}

// Google CSE search
type googleSearchResponse struct {
	Items []struct {
		Title   string `json:"title"`
		Link    string `json:"link"`
		Snippet string `json:"snippet"`
	} `json:"items"`
}

// GoogleSearch uses Google Custom Search JSON API. Provide apiKey and cseId.
func GoogleSearch(query, apiKey, cseId string, topK int) ([]SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("empty query")
	}
	if apiKey == "" || cseId == "" {
		return nil, errors.New("google api key or cse id missing")
	}
	if topK <= 0 {
		topK = 3
	}
	base := "https://www.googleapis.com/customsearch/v1"
	u, _ := url.Parse(base)
	q := u.Query()
	q.Set("key", apiKey)
	q.Set("cx", cseId)
	q.Set("q", query)
	// increase candidates to improve recall
	want := topK * 3
	if want < topK {
		want = topK
	}
	if want > 10 {
		want = 10
	}
	q.Set("num", strconv.Itoa(want))
	// language/region hints
	if containsChinese(query) {
		q.Set("hl", "zh-CN")
		q.Set("gl", "CN")
		q.Set("lr", "lang_zh-CN")
	} else {
		q.Set("hl", "en")
		q.Set("gl", "US")
	}
	u.RawQuery = q.Encode()

	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", u.String(), nil)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New("google search failed: " + resp.Status)
	}
	body, _ := ioutil.ReadAll(resp.Body)
	var data googleSearchResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	var out []SearchResult
	for _, it := range data.Items {
		if it.Link == "" {
			continue
		}
		out = append(out, SearchResult{Title: it.Title, URL: it.Link, Snippet: it.Snippet})
		if len(out) >= topK {
			break
		}
	}
	if len(out) == 0 {
		return nil, errors.New("no results")
	}
	return out, nil
}

// BuildGoogleSearchContext uses Google CSE to find results and then fetches their content via reader
func BuildGoogleSearchContext(query, apiKey, cseId string, topK int) (string, error) {
	results, err := GoogleSearch(query, apiKey, cseId, topK)
	if err != nil {
		return "", err
	}
	type item struct {
		Title   string `json:"title"`
		URL     string `json:"url"`
		Snippet string `json:"snippet,omitempty"`
		Content string `json:"content"`
	}
	var items []item
	for _, r := range results {
		content, err := FetchURLAsPlainText(r.URL)
		if err != nil {
			continue
		}
		// trim very long content to reduce token waste
		content = trimTo(content, 4000)
		items = append(items, item{Title: r.Title, URL: r.URL, Snippet: r.Snippet, Content: content})
	}
	if len(items) == 0 {
		return "", errors.New("no accessible results")
	}
	b, _ := json.Marshal(items)
	return string(b), nil
}

// helpers
func trimTo(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	return s[:n]
}

var zhChar = regexp.MustCompile(`[\p{Han}]`)

func containsChinese(s string) bool {
	return zhChar.MatchString(s)
}
