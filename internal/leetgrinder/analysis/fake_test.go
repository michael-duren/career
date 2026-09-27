package analysis

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

const testKey = "sk-ant-test-secret-key-0123456789"

// fakeAPI is an httptest stand-in for the Messages API. It records requests
// and answers with the configured reply.
type fakeAPI struct {
	*httptest.Server
	mu       sync.Mutex
	requests []fakeRequest
	status   int
	// body is the response body; it defaults to a message with text.
	body string
	text string
	stop string
}

type fakeRequest struct {
	Path   string
	Header http.Header
	Body   map[string]any
}

func newFakeAPI(t *testing.T) *fakeAPI {
	f := &fakeAPI{status: 200, stop: "end_turn", text: `{"actualTime":"O(n)","actualSpace":"O(n)","timeMatches":true,"spaceMatches":false,"optimal":true,"explanation":"One pass with a hash map."}`}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		f.mu.Lock()
		f.requests = append(f.requests, fakeRequest{r.URL.Path, r.Header.Clone(), body})
		status, reply, text, stop := f.status, f.body, f.text, f.stop
		f.mu.Unlock()
		if reply == "" {
			content := []map[string]any{{"type": "thinking", "thinking": "", "signature": "sig"}, {"type": "text", "text": text}}
			b, _ := json.Marshal(map[string]any{"id": "msg_test", "type": "message", "role": "assistant", "model": "claude-sonnet-5", "content": content, "stop_reason": stop, "stop_sequence": nil, "usage": map[string]any{"input_tokens": 10, "output_tokens": 20}})
			reply = string(b)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(f.Close)
	return f
}

func (f *fakeAPI) reply(status int, body string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status, f.body = status, body
}

func (f *fakeAPI) answer(text, stop string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status, f.body, f.text, f.stop = 200, "", text, stop
}

func (f *fakeAPI) take() []fakeRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := f.requests
	f.requests = nil
	return out
}
