package implassistant

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SigNoz/signoz/pkg/assistant"
)

type memoryStore struct {
	configs map[string]assistant.Config
}

func newMemoryStore() *memoryStore {
	return &memoryStore{configs: make(map[string]assistant.Config)}
}

func (store *memoryStore) GetConfig(_ context.Context, orgID string) (*assistant.Config, error) {
	config, ok := store.configs[orgID]
	if !ok {
		return nil, assistant.ErrConfigNotFound
	}
	return &config, nil
}

func (store *memoryStore) UpsertConfig(_ context.Context, orgID string, config assistant.Config) error {
	store.configs[orgID] = config
	return nil
}

func TestUpdateConfigPreservesConfiguredAPIKey(t *testing.T) {
	store := newMemoryStore()
	module := NewModule(store, nil)
	apiKey := "first-key"

	err := module.UpdateConfig(context.Background(), "org-1", assistant.UpdatableConfig{
		BaseURL: "https://api.openai.com/v1/",
		Model:   "gpt-4o-mini",
		APIKey:  &apiKey,
	})
	if err != nil {
		t.Fatalf("update initial config: %v", err)
	}

	err = module.UpdateConfig(context.Background(), "org-1", assistant.UpdatableConfig{
		BaseURL: "http://localhost:11434/v1",
		Model:   "llama3.2",
	})
	if err != nil {
		t.Fatalf("update config without key: %v", err)
	}

	stored := store.configs["org-1"]
	if stored.APIKey != apiKey {
		t.Fatalf("expected API key to be preserved, got %q", stored.APIKey)
	}
	if stored.BaseURL != "http://localhost:11434/v1" {
		t.Fatalf("expected normalized base URL, got %q", stored.BaseURL)
	}
}

func TestChatStreamsOpenAICompatibleResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected authorization header %q", got)
		}

		request := new(openAIRequest)
		if err := json.NewDecoder(r.Body).Decode(request); err != nil {
			t.Fatalf("decode upstream request: %v", err)
		}
		if !request.Stream || request.Model != "test-model" {
			t.Fatalf("unexpected upstream request %+v", request)
		}
		if len(request.Messages) != 2 || request.Messages[0].Role != "system" {
			t.Fatalf("expected system and user messages, got %+v", request.Messages)
		}

		rw.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(rw, "data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n")
		_, _ = fmt.Fprint(rw, "data: {\"choices\":[{\"delta\":{\"content\":\" world\"}}]}\n\n")
		_, _ = fmt.Fprint(rw, "data: [DONE]\n\n")
	}))
	defer server.Close()

	store := newMemoryStore()
	store.configs["org-1"] = assistant.Config{
		BaseURL: server.URL,
		Model:   "test-model",
		APIKey:  "test-key",
	}
	module := NewModule(store, server.Client())
	var response strings.Builder

	err := module.Chat(context.Background(), "org-1", assistant.ChatRequest{
		Messages: []assistant.ChatMessage{{Role: "user", Content: "hello"}},
		Context:  assistant.ContextSnapshot{Route: "/logs"},
	}, func(event assistant.StreamEvent) error {
		response.WriteString(event.Delta)
		return nil
	})
	if err != nil {
		t.Fatalf("stream chat: %v", err)
	}
	if response.String() != "Hello world" {
		t.Fatalf("unexpected streamed response %q", response.String())
	}
}
