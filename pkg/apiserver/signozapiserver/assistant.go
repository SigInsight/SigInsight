package signozapiserver

import (
	"net/http"

	"github.com/SigNoz/signoz/pkg/assistant"
	"github.com/SigNoz/signoz/pkg/http/handler"
	"github.com/SigNoz/signoz/pkg/types"
	"github.com/gorilla/mux"
)

func (provider *provider) addAssistantRoutes(router *mux.Router) error {
	if err := router.Handle("/api/v1/assistant/config", handler.New(provider.authZ.ViewAccess(provider.assistantHandler.GetConfig), handler.OpenAPIDef{
		ID:                  "GetAssistantConfig",
		Tags:                []string{"assistant"},
		Summary:             "Get AI assistant configuration",
		Description:         "Returns the configured OpenAI-compatible endpoint without exposing its API key",
		Response:            new(assistant.ConfigResponse),
		ResponseContentType: "application/json",
		SuccessStatusCode:   http.StatusOK,
		SecuritySchemes:     newSecuritySchemes(types.RoleViewer),
	})).Methods(http.MethodGet).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v1/assistant/config", handler.New(provider.authZ.AdminAccess(provider.assistantHandler.UpdateConfig), handler.OpenAPIDef{
		ID:                 "UpdateAssistantConfig",
		Tags:               []string{"assistant"},
		Summary:            "Update AI assistant configuration",
		Description:        "Stores an OpenAI-compatible endpoint configuration for the organization",
		Request:            new(assistant.UpdatableConfig),
		RequestContentType: "application/json",
		SuccessStatusCode:  http.StatusNoContent,
		SecuritySchemes:    newSecuritySchemes(types.RoleAdmin),
	})).Methods(http.MethodPut).GetError(); err != nil {
		return err
	}

	if err := router.Handle("/api/v1/assistant/chat", handler.New(provider.authZ.ViewAccess(provider.assistantHandler.Chat), handler.OpenAPIDef{
		ID:                  "ChatWithAssistant",
		Tags:                []string{"assistant-stream"},
		Summary:             "Chat with AI assistant",
		Description:         "Streams Server-Sent Events. Token events contain a delta, error events contain a message, and the stream ends with a done event.",
		Request:             new(assistant.ChatRequest),
		RequestContentType:  "application/json",
		Response:            new(assistant.StreamEvent),
		ResponseContentType: "text/event-stream",
		ResponseIsRaw:       true,
		SuccessStatusCode:   http.StatusOK,
		ErrorStatusCodes:    []int{http.StatusBadRequest},
		SecuritySchemes:     newSecuritySchemes(types.RoleViewer),
	})).Methods(http.MethodPost).GetError(); err != nil {
		return err
	}

	return nil
}
