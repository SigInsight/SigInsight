package implassistant

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/SigNoz/signoz/pkg/assistant"
	"github.com/SigNoz/signoz/pkg/errors"
)

const maxMessages = 20

type module struct {
	store      assistant.Store
	httpClient *http.Client
}

func NewModule(store assistant.Store, httpClient *http.Client) assistant.Module {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 90 * time.Second}
	}

	return &module{store: store, httpClient: httpClient}
}

func (module *module) GetConfig(ctx context.Context, orgID string) (*assistant.ConfigResponse, error) {
	config, err := module.store.GetConfig(ctx, orgID)
	if err != nil {
		return nil, err
	}

	response := &assistant.ConfigResponse{
		BaseURL: assistant.DefaultBaseURL,
		Model:   assistant.DefaultModel,
	}
	if config == nil {
		return response, nil
	}

	response.BaseURL = config.BaseURL
	response.Model = config.Model
	response.APIKeyConfigured = config.APIKey != ""
	return response, nil
}

func (module *module) UpdateConfig(ctx context.Context, orgID string, input assistant.UpdatableConfig) error {
	baseURL, err := normalizeBaseURL(input.BaseURL)
	if err != nil {
		return err
	}

	model := strings.TrimSpace(input.Model)
	if model == "" {
		return errors.New(errors.TypeInvalidInput, errors.CodeInvalidInput, "model is required")
	}

	existing, err := module.store.GetConfig(ctx, orgID)
	if err != nil {
		return err
	}

	apiKey := ""
	if existing != nil {
		apiKey = existing.APIKey
	}
	if input.APIKey != nil {
		apiKey = strings.TrimSpace(*input.APIKey)
	}
	if apiKey == "" {
		return errors.New(errors.TypeInvalidInput, errors.CodeInvalidInput, "API key is required")
	}

	return module.store.UpsertConfig(ctx, orgID, assistant.Config{
		BaseURL: baseURL,
		Model:   model,
		APIKey:  apiKey,
	})
}

func (module *module) Chat(ctx context.Context, orgID string, request assistant.ChatRequest, emit func(assistant.StreamEvent) error) error {
	if err := validateChatRequest(request); err != nil {
		return err
	}

	config, err := module.store.GetConfig(ctx, orgID)
	if err != nil {
		return err
	}
	if config == nil || config.APIKey == "" {
		return errors.New(errors.TypeInvalidInput, errors.CodeInvalidInput, "an LLM API key has not been configured")
	}

	requestBody, err := newOpenAIRequest(config, request)
	if err != nil {
		return err
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, config.BaseURL+"/chat/completions", bytes.NewReader(requestBody))
	if err != nil {
		return errors.Wrap(err, errors.TypeInternal, errors.CodeUnknown, "create LLM request")
	}
	httpRequest.Header.Set("Authorization", "Bearer "+config.APIKey)
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "text/event-stream")

	response, err := module.httpClient.Do(httpRequest)
	if err != nil {
		return errors.Wrap(err, errors.TypeUnexpected, errors.CodeUnknown, "LLM request failed")
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = response.Status
		}
		return errors.Newf(errors.TypeUnexpected, errors.CodeUnknown, "LLM API request failed: %s", message)
	}

	return streamOpenAIResponse(response.Body, emit)
}

func normalizeBaseURL(value string) (string, error) {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New(errors.TypeInvalidInput, errors.CodeInvalidInput, "base URL must be an HTTP(S) URL without query parameters")
	}

	return value, nil
}

func validateChatRequest(request assistant.ChatRequest) error {
	if len(request.Messages) == 0 {
		return errors.New(errors.TypeInvalidInput, errors.CodeInvalidInput, "at least one message is required")
	}
	if len(request.Messages) > maxMessages {
		return errors.Newf(errors.TypeInvalidInput, errors.CodeInvalidInput, "at most %d messages are allowed", maxMessages)
	}

	for _, message := range request.Messages {
		if message.Role != "user" && message.Role != "assistant" {
			return errors.New(errors.TypeInvalidInput, errors.CodeInvalidInput, "messages must have user or assistant roles")
		}
		if strings.TrimSpace(message.Content) == "" {
			return errors.New(errors.TypeInvalidInput, errors.CodeInvalidInput, "message content is required")
		}
	}

	return nil
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

func newOpenAIRequest(config *assistant.Config, request assistant.ChatRequest) ([]byte, error) {
	contextJSON, err := json.Marshal(request.Context)
	if err != nil {
		return nil, errors.Wrap(err, errors.TypeInvalidInput, errors.CodeInvalidInput, "serialize AI assistant context")
	}

	messages := make([]openAIMessage, 0, len(request.Messages)+1)
	messages = append(messages, openAIMessage{
		Role: "system",
		Content: "You are the SigInsight AI assistant. Answer using the user's question and the UI context below. " +
			"Do not claim that you queried telemetry data or changed a query unless a tool result explicitly appears in the conversation. " +
			"Be clear about uncertainty.\n\nUI context:\n" + string(contextJSON),
	})
	for _, message := range request.Messages {
		messages = append(messages, openAIMessage{Role: message.Role, Content: message.Content})
	}

	return json.Marshal(openAIRequest{Model: config.Model, Messages: messages, Stream: true})
}

type openAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

func streamOpenAIResponse(reader io.Reader, emit func(assistant.StreamEvent) error) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			return nil
		}

		chunk := new(openAIStreamChunk)
		if err := json.Unmarshal([]byte(payload), chunk); err != nil {
			return errors.Wrap(err, errors.TypeUnexpected, errors.CodeUnknown, "LLM API returned an invalid stream event")
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content == "" {
				continue
			}
			if err := emit(assistant.StreamEvent{Type: "token", Delta: choice.Delta.Content}); err != nil {
				return fmt.Errorf("write assistant stream: %w", err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return errors.Wrap(err, errors.TypeUnexpected, errors.CodeUnknown, "read LLM response stream")
	}

	return nil
}
