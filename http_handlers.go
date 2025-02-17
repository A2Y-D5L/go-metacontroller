package metacontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/a2y-d5l/go-metacontroller/composition"
)

type (
	// rawCompositeRequest mirrors the JSON payload for the sync hook.
	rawCompositeRequest struct {
		Parent     json.RawMessage                       `json:"parent"`
		Children   map[string]map[string]json.RawMessage `json:"children,omitempty"`
		Finalizing bool                                  `json:"finalizing"`
	}

	// rawCompositeResponse is used to encode the sync hook response.
	rawCompositeResponse struct {
		Status    json.RawMessage              `json:"status,omitempty"`
		Children  map[string][]json.RawMessage `json:"children,omitempty"`
		Finalized bool                         `json:"finalized,omitempty"`
	}

	// rawCustomizeRequest mirrors the JSON payload for the customize hook.
	rawCustomizeRequest struct {
		Controller json.RawMessage `json:"controller"`
		Parent     json.RawMessage `json:"parent"`
	}
)

// writeError logs an error and writes an HTTP error response. If debug is true, the detailed error message is exposed in the response.
func writeError(ctx context.Context, w http.ResponseWriter, code int, err error, logger *slog.Logger) {
	slog.Error("Error: " + err.Error())
	var msg string
	switch code {
	case http.StatusBadRequest:
		msg = "bad request"
	case http.StatusInternalServerError:
		msg = "internal server error"
	case http.StatusMethodNotAllowed:
		msg = "method not allowed"
	default:
		msg = http.StatusText(code)
	}

	if logger.Enabled(ctx, slog.LevelDebug) {
		msg = fmt.Sprintf("%s: %v", msg, err)
	}
	http.Error(w, msg, code)
}

// syncHandler handles sync hook HTTP requests.
type syncHandler[P client.Object] struct {
	scheme  *runtime.Scheme
	decoder runtime.Decoder
	encoder runtime.Encoder
	syncer  composition.Syncer[P]
	logger  *slog.Logger
}

// ServeHTTP processes sync hook HTTP requests.
func (sh *syncHandler[P]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var rawReq rawCompositeRequest
	if err := json.NewDecoder(r.Body).Decode(&rawReq); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, fmt.Errorf("SyncHook: error decoding request: %w", err), sh.logger)

		return
	}

	p, _, err := sh.decoder.Decode(rawReq.Parent, nil, nil)
	if err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, fmt.Errorf("SyncHook: error decoding parent: %w", err), sh.logger)

		return
	}

	parent, ok := p.(P)
	if !ok {
		writeError(r.Context(), w, http.StatusBadRequest, fmt.Errorf("SyncHook: type assertion failure: parent"), sh.logger)

		return
	}

	observedChildren := make(map[schema.GroupVersionKind][]client.Object)
	for _, rawList := range rawReq.Children {
		for _, rawChild := range rawList {
			childObj, childGVK, err := sh.decoder.Decode(rawChild, nil, nil)
			if err != nil {
				sh.logger.ErrorContext(r.Context(),
					"SyncHook: error decoding child",
					"error", err.Error(),
					"child", string(rawChild))

				continue
			}

			child, ok := childObj.(client.Object)
			if !ok {
				sh.logger.ErrorContext(r.Context(),
					"SyncHook: type assertion failure: child is not a client.Object",
					"child",
					string(rawChild))

				continue
			}
			observedChildren[*childGVK] = append(observedChildren[*childGVK], child)
		}
	}

	resp, err := sh.syncer.Sync(r.Context(), sh.scheme, &composition.SyncRequest[P]{
		Parent:     parent,
		Children:   observedChildren,
		Finalizing: rawReq.Finalizing,
	})
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, fmt.Errorf("SyncHook: handler error: %w", err), sh.logger)

		return
	}

	statusBytes, err := runtime.Encode(sh.encoder, resp.Status)
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, fmt.Errorf("SyncHook: error encoding status: %w", err), sh.logger)

		return
	}

	desiredChildren := make(map[string][]json.RawMessage)
	for gvk, objs := range resp.Children {
		key := KeyForGVK(gvk)
		var rawList []json.RawMessage
		for _, obj := range objs {
			data, err := runtime.Encode(sh.encoder, obj)
			if err != nil {
				writeError(r.Context(), w, http.StatusInternalServerError, fmt.Errorf("SyncHook: error encoding child: %w", err), sh.logger)

				return
			}

			rawList = append(rawList, json.RawMessage(data))
		}
		desiredChildren[key] = rawList
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(rawCompositeResponse{
		Status:   statusBytes,
		Children: desiredChildren,
	}); err != nil {
		sh.logger.ErrorContext(r.Context(), "SyncHook: error encoding response: "+err.Error())
	}
}

type customizeHandler[P client.Object] struct {
	scheme     *runtime.Scheme
	decoder    runtime.Decoder
	customizer composition.Customizer[P]
	logger     *slog.Logger
}

// ServeHTTP processes customize hook HTTP requests.
func (ch *customizeHandler[P]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var rawReq rawCustomizeRequest
	if err := json.NewDecoder(r.Body).Decode(&rawReq); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, fmt.Errorf("CustomizeHook: error decoding request: %w", err), ch.logger)
		return
	}

	p, _, err := ch.decoder.Decode(rawReq.Parent, nil, nil)
	if err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, fmt.Errorf("CustomizeHook: error decoding parent: %w", err), ch.logger)
		return
	}

	parent, ok := p.(P)
	if !ok {
		writeError(r.Context(), w, http.StatusBadRequest, fmt.Errorf("CustomizeHook: type assertion failure for parent"), ch.logger)
		return
	}

	resp, err := ch.customizer.Customize(r.Context(), ch.scheme, &composition.CustomizeRequest[P]{
		Controller: rawReq.Controller,
		Parent:     parent,
	})
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, fmt.Errorf("CustomizeHook: CustomizeHandler failed with error: %w", err), ch.logger)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		ch.logger.Error("CustomizeHook: error encoding response", "error", err.Error())
	}
}

type finalizeHandler[P client.Object] struct {
	scheme    *runtime.Scheme
	decoder   runtime.Decoder
	finalizer composition.Finalizer[P]
	logger    *slog.Logger
}

// ServeHTTP processes finalize hook HTTP requests.
func (fh *finalizeHandler[P]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var rawReq rawCompositeRequest
	if err := json.NewDecoder(r.Body).Decode(&rawReq); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, fmt.Errorf("FinalizeHook: error decoding request: %w", err), fh.logger)
		return
	}

	p, _, err := fh.decoder.Decode(rawReq.Parent, nil, nil)
	if err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, fmt.Errorf("FinalizeHook: error decoding parent: %w", err), fh.logger)
		return
	}

	parent, ok := p.(P)
	if !ok {
		writeError(r.Context(), w, http.StatusBadRequest, fmt.Errorf("FinalizeHook: type assertion failure for parent"), fh.logger)
		return
	}

	observedChildren := make(map[schema.GroupVersionKind][]client.Object)
	for _, rawList := range rawReq.Children {
		for _, rawChild := range rawList {
			childObj, childGVK, err := fh.decoder.Decode(rawChild, nil, nil)
			if err != nil {
				fh.logger.ErrorContext(r.Context(),
					"Finalize error: unable to decoding child",
					"error", err.Error(),
					"child", string(rawChild))

				continue
			}

			child, ok := childObj.(client.Object)
			if !ok {
				fh.logger.ErrorContext(r.Context(),
					"Finalize error: child is not a client.Object",
					"child",
					string(rawChild))

				continue
			}
			observedChildren[*childGVK] = append(observedChildren[*childGVK], child)
		}
	}

	resp, err := fh.finalizer.Finalize(r.Context(), fh.scheme, &composition.FinalizeRequest[P]{
		Parent: parent,
	})
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError,
			fmt.Errorf("FinalizeHook: FinalizeHandler failed with error: %w", err),
			fh.logger)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		fh.logger.Error("Finalize error: unable to encode response", "error", err.Error())
	}
}
