package implassistant

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/SigNoz/signoz/pkg/assistant"
	"github.com/SigNoz/signoz/pkg/http/render"
	"github.com/SigNoz/signoz/pkg/types/authtypes"
)

type handler struct {
	module assistant.Module
}

func NewHandler(module assistant.Module) assistant.Handler {
	return &handler{module: module}
}

func (handler *handler) GetConfig(rw http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	claims, err := authtypes.ClaimsFromContext(ctx)
	if err != nil {
		render.Error(rw, err)
		return
	}

	config, err := handler.module.GetConfig(ctx, claims.OrgID)
	if err != nil {
		render.Error(rw, err)
		return
	}

	render.Success(rw, http.StatusOK, config)
}

func (handler *handler) UpdateConfig(rw http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	claims, err := authtypes.ClaimsFromContext(ctx)
	if err != nil {
		render.Error(rw, err)
		return
	}

	input := new(assistant.UpdatableConfig)
	decoder := json.NewDecoder(http.MaxBytesReader(rw, r.Body, 64*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(input); err != nil {
		render.Error(rw, err)
		return
	}

	if err := handler.module.UpdateConfig(ctx, claims.OrgID, *input); err != nil {
		render.Error(rw, err)
		return
	}

	render.Success(rw, http.StatusNoContent, nil)
}

func (handler *handler) Chat(rw http.ResponseWriter, r *http.Request) {
	claims, err := authtypes.ClaimsFromContext(r.Context())
	if err != nil {
		render.Error(rw, err)
		return
	}

	request := new(assistant.ChatRequest)
	decoder := json.NewDecoder(http.MaxBytesReader(rw, r.Body, 256*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(request); err != nil {
		render.Error(rw, err)
		return
	}

	flusher, ok := rw.(http.Flusher)
	if !ok {
		render.Error(rw, fmt.Errorf("streaming is not supported by the response writer"))
		return
	}

	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Connection", "keep-alive")
	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("X-Accel-Buffering", "no")
	rw.WriteHeader(http.StatusOK)
	flusher.Flush()

	emit := func(event assistant.StreamEvent) error {
		return writeSSE(rw, flusher, event.Type, event)
	}
	if err := handler.module.Chat(r.Context(), claims.OrgID, *request, emit); err != nil {
		_ = writeSSE(rw, flusher, "error", assistant.StreamEvent{Message: err.Error()})
		return
	}

	_ = writeSSE(rw, flusher, "done", struct{}{})
}

func writeSSE(rw http.ResponseWriter, flusher http.Flusher, event string, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(rw, "event: %s\ndata: %s\n\n", event, payload); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}
