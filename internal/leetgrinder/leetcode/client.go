// Package leetcode fetches problem metadata from LeetCode's public GraphQL
// endpoint, one slug at a time, for problems the extension did not describe.
package leetcode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

// Endpoint is LeetCode's GraphQL API.
const Endpoint = "https://leetcode.com/graphql"

// Timeout bounds one request.
const Timeout = 10 * time.Second

// maxResponseBytes bounds the body read from LeetCode.
const maxResponseBytes = 256 << 10

const query = `query questionData($titleSlug: String!) { question(titleSlug: $titleSlug) { questionFrontendId title difficulty topicTags { slug name } } }`

// Client asks LeetCode about one problem.
type Client struct {
	// URL is Endpoint unless a test overrides it.
	URL  string
	HTTP *http.Client
}

func (c *Client) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: Timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

// Fetch returns slug's metadata, or nil when LeetCode answers that no such
// problem exists. Any other failure is an error.
func (c *Client) Fetch(ctx context.Context, slug string) (*leetgrinder.ProblemMetadata, error) {
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	body, err := json.Marshal(map[string]any{"operationName": "questionData", "query": query, "variables": map[string]string{"titleSlug": slug}})
	if err != nil {
		return nil, err
	}
	url := c.URL
	if url == "" {
		url = Endpoint
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", "https://leetcode.com/problems/"+slug+"/")
	res, err := c.http().Do(req)
	if err != nil {
		return nil, errors.New("request failed")
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LeetCode answered with status %d", res.StatusCode)
	}
	var out struct {
		Data *struct {
			Question *struct {
				QuestionFrontendID string                 `json:"questionFrontendId"`
				Title              string                 `json:"title"`
				Difficulty         string                 `json:"difficulty"`
				TopicTags          []leetgrinder.TopicTag `json:"topicTags"`
			} `json:"question"`
		} `json:"data"`
	}
	if err = json.NewDecoder(io.LimitReader(res.Body, maxResponseBytes)).Decode(&out); err != nil || out.Data == nil {
		return nil, errors.New("unexpected response")
	}
	q := out.Data.Question
	if q == nil {
		return nil, nil
	}
	m := leetgrinder.ProblemMetadata{Title: q.Title, Difficulty: q.Difficulty, Topics: q.TopicTags}
	// Frontend ids are numeric for problems; anything else stays unknown.
	if n, err := strconv.Atoi(q.QuestionFrontendID); err == nil {
		m.Number = n
	}
	if m.Normalize() != nil || m.Title == "" {
		return nil, errors.New("invalid metadata")
	}
	return &m, nil
}
