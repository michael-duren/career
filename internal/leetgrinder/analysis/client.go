// Package analysis asks an LLM, through the Anthropic Messages API, for the
// correctness reasoning and complexity of saved Leetgrinder attempt code.
package analysis

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

const (
	// Timeout bounds one request.
	Timeout = 5 * time.Minute
	// MaxTokens caps the output, including adaptive thinking, of one request.
	MaxTokens = 16000
)

// Input is one attempt to analyse.
type Input struct {
	Correctness leetgrinder.Correctness
	// Problem is the catalog entry; it may have only Slug set.
	Problem     leetgrinder.Problem
	Language    string
	Code        string
	StatedTime  string
	StatedSpace string
}

// Error is a failed request. Its message is safe to store and show: it never
// contains the API key, the request body, or the response body.
type Error struct {
	Message string
	// Retry reports whether a later request may succeed.
	Retry bool
	// Pause asks the worker to stop sending for a while: the error is
	// likely to hit every attempt (configuration, account, or a request
	// shape the model rejects).
	Pause bool
	// Config marks a certain configuration problem (key, permissions,
	// billing, or model); it does not count as a try for the attempt.
	Config bool
}

func (e *Error) Error() string { return e.Message }

// Reference reports whether the problem has a known optimum to compare
// with. Without one the model estimates it.
func (in Input) Reference() bool { return in.Problem.HasOptimal() }

// Client calls the Messages API with one model.
type Client struct {
	api   anthropic.Client
	model string
	key   string
}

// NewClient returns a client for key and model. baseURL and httpClient are
// optional and exist for tests. The SDK's environment defaults are disabled,
// so only key is ever sent, and the SDK does not retry: the worker counts and
// spaces every request itself.
func NewClient(key, model, baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: Timeout + 10*time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	opts := []option.RequestOption{
		option.WithoutEnvironmentDefaults(),
		option.WithAPIKey(key),
		option.WithHTTPClient(httpClient),
		option.WithMaxRetries(0),
		option.WithRequestTimeout(Timeout),
	}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	return &Client{api: anthropic.NewClient(opts...), model: model, key: key}
}

// Model is the model the client asks.
func (c *Client) Model() string { return c.model }

// Analyze requests one analysis and validates the structured response.
func (c *Client) Analyze(ctx context.Context, in Input) (leetgrinder.AnalysisResult, error) {
	boundary, err := newBoundary()
	if err != nil {
		return leetgrinder.AnalysisResult{}, &Error{Message: "could not build the request", Retry: true}
	}
	adaptive := anthropic.ThinkingConfigAdaptiveParam{}
	msg, err := c.api.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(c.model),
		MaxTokens: MaxTokens,
		System:    []anthropic.TextBlockParam{{Text: systemPrompt}},
		Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt(in, boundary)))},
		Thinking:  anthropic.ThinkingConfigParamUnion{OfAdaptive: &adaptive},
		OutputConfig: anthropic.OutputConfigParam{
			Effort: anthropic.OutputConfigEffortMedium,
			Format: anthropic.JSONOutputFormatParam{Schema: in.schema()},
		},
	})
	if err != nil {
		return leetgrinder.AnalysisResult{}, c.requestError(err)
	}
	switch msg.StopReason {
	case anthropic.StopReasonEndTurn:
	case anthropic.StopReasonRefusal:
		return leetgrinder.AnalysisResult{}, &Error{Message: "the model declined to analyse this code"}
	case anthropic.StopReasonMaxTokens:
		return leetgrinder.AnalysisResult{}, &Error{Message: "the response was cut off at the output token limit"}
	default:
		return leetgrinder.AnalysisResult{}, &Error{Message: "the response ended unexpectedly (" + safeWord(string(msg.StopReason)) + ")", Retry: true}
	}
	var text strings.Builder
	for _, block := range msg.Content {
		if b, ok := block.AsAny().(anthropic.TextBlock); ok {
			text.WriteString(b.Text)
		}
	}
	result, err := ParseResult(text.String(), in)
	if err != nil {
		return result, &Error{Message: "the response was invalid: " + err.Error(), Retry: true}
	}
	return result, nil
}

// requestError maps an SDK error onto a safe summary. Only the status code
// and error type of an API error are kept, never its body or request.
func (c *Client) requestError(err error) *Error {
	var apiErr *anthropic.Error
	if errors.As(err, &apiErr) {
		msg := fmt.Sprintf("Anthropic API returned %d", apiErr.StatusCode)
		if t := safeWord(string(apiErr.Type())); t != "" {
			msg += " " + t
		}
		switch apiErr.StatusCode {
		case http.StatusUnauthorized:
			msg += ": check ANTHROPIC_API_KEY"
		case http.StatusNotFound:
			msg += ": check LEETGRINDER_ANALYSIS_MODEL"
		}
		switch apiErr.StatusCode {
		case http.StatusUnauthorized, http.StatusPaymentRequired, http.StatusForbidden, http.StatusNotFound:
			return &Error{Message: msg, Retry: true, Pause: true, Config: true}
		case http.StatusBadRequest:
			// Inputs are bounded, so a 400 usually means the model or account
			// cannot serve this request shape (or credit ran out). It still
			// counts as a try, so one bad attempt cannot block the queue.
			return &Error{Message: msg, Retry: true, Pause: true}
		}
		retry := apiErr.StatusCode == http.StatusRequestTimeout || apiErr.StatusCode == http.StatusConflict || apiErr.StatusCode == http.StatusTooManyRequests || apiErr.StatusCode >= 500
		return &Error{Message: msg, Retry: retry}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &Error{Message: "Anthropic API request timed out", Retry: true}
	}
	if errors.Is(err, context.Canceled) {
		return &Error{Message: "Anthropic API request was cancelled", Retry: true}
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Timeout() {
			return &Error{Message: "Anthropic API request timed out", Retry: true}
		}
		err = urlErr.Err
	}
	return &Error{Message: Redact("Anthropic API request failed: "+err.Error(), c.key), Retry: true}
}

// Redact removes the key from s.
func Redact(s, key string) string {
	if key == "" {
		return s
	}
	return strings.ReplaceAll(s, key, "[redacted]")
}

// safeWord keeps an identifier such as an error type short and printable.
func safeWord(s string) string {
	s = strings.Map(func(r rune) rune {
		if r == '_' || r == '-' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, s)
	if len(s) > 40 {
		s = s[:40]
	}
	return s
}

func newBoundary() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
