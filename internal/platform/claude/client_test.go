package claude

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStructuredRetriesAndForcesTool(t *testing.T) {
	var got map[string]any
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("x-api-key") != "k" || r.Header.Get("anthropic-version") == "" {
			t.Errorf("missing auth/version headers")
		}
		if calls == 1 { // overloaded once: client must retry
			w.WriteHeader(529)
			w.Write([]byte(`{"type":"error","error":{"type":"overloaded_error","message":"busy"}}`))
			return
		}
		json.NewDecoder(r.Body).Decode(&got)
		w.Write([]byte(`{"model":"m","usage":{"input_tokens":10,"output_tokens":5,"cache_read_input_tokens":3},
			"content":[{"type":"tool_use","id":"t","name":"echo","input":{"value":42}}]}`))
	}))
	defer srv.Close()
	SetBaseURL(srv.URL)

	var out struct{ Value int }
	res, err := NewClient("k").Structured(context.Background(),
		Request{Model: "m", MaxTokens: 10, Messages: []Message{{Role: "user", Content: []Block{{Type: "text", Text: "hi"}}}}},
		Tool{Name: "echo", InputSchema: Obj(Schema{"value": Schema{"type": "integer"}})}, &out)
	if err != nil {
		t.Fatal(err)
	}
	if out.Value != 42 || calls != 2 || res.Usage.CacheReadInputTokens != 3 {
		t.Fatalf("out=%+v calls=%d usage=%+v", out, calls, res.Usage)
	}
	if tc := got["tool_choice"].(map[string]any); tc["type"] != "tool" || tc["name"] != "echo" {
		t.Fatalf("tool_choice = %v", tc)
	}
}

func TestMissingKey(t *testing.T) {
	if _, err := NewClient("").Create(context.Background(), Request{}); err != ErrNoAPIKey {
		t.Fatalf("want ErrNoAPIKey, got %v", err)
	}
}
