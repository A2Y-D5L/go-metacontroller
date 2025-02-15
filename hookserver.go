// Package metacontroller provides a webhook server framework for implementing
// CompositeController hooks for Metacontroller. Consumers can run multiple hooks
// (for various parent resource types) by supplying sync and/or customize handlers
// via functional options. The HookServer creates its own HTTP multiplexer so that it
// isn’t bound to the default HTTP server.
package metacontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// --- Logger and Helper Functions ---

// Logger defines the interface that a logger should implement.
type Logger interface {
	Printf(format string, v ...interface{})
}

// defaultLogger is a simple logger that wraps the standard log.Printf.
type defaultLogger struct{}

// Printf logs the formatted message using the standard log package.
func (d *defaultLogger) Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

// writeError logs an error and writes an HTTP error response. If debug is true, the detailed error message is exposed in the response.
func writeError(w http.ResponseWriter, code int, err error, logger Logger, debug bool) {
	logger.Printf("Error: %v", err)
	var msg string
	if debug {
		msg = err.Error()
	} else {
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
	}
	http.Error(w, msg, code)
}

// --- Internal types for the sync hook ---

// rawCompositeRequest mirrors the JSON payload for the sync hook.
type rawCompositeRequest struct {
	Parent		json.RawMessage              `json:"parent"`
	Children	map[string][]json.RawMessage `json:"children"`
	Finalizing	bool                         `json:"finalizing"`
}

// rawCompositeResponse is used to encode the sync hook response.
type rawCompositeResponse struct {
	Status   json.RawMessage              `json:"status,omitempty"`
	Children map[string][]json.RawMessage `json:"children,omitempty"`
}

// CompositeRequest represents the fully decoded sync hook request.
type CompositeRequest[TParent runtime.Object] struct {
	// Parent is the composite (parent) resource.
	Parent TParent
	// Children is a map from GroupVersionKind to slices of decoded child objects.
	Children map[schema.GroupVersionKind][]runtime.Object
	// Finalizing indicates the type of sync operation (sync=false, finalize=true).
	Finalizing bool
}

// CompositeResponse represents the sync hook response.
type CompositeResponse[TParent runtime.Object] struct {
	// Status is the updated composite (parent) resource.
	Status TParent
	// Children defines the desired state for child objects.
	Children map[schema.GroupVersionKind][]runtime.Object
}

// SyncHandler is a function type for processing decoded sync hook requests. It receives a context, the runtime scheme, and a decoded composite request, then returns a decoded composite response or an error.
type SyncHandler[TParent runtime.Object] func(ctx context.Context, scheme *runtime.Scheme, req *CompositeRequest[TParent]) (*CompositeResponse[TParent], error)

// --- Types for the customize hook ---

// rawCustomizeRequest mirrors the JSON payload for the customize hook.
type rawCustomizeRequest struct {
	Controller json.RawMessage `json:"controller"`
	Parent     json.RawMessage `json:"parent"`
}

// CustomizeRequest represents the customize hook request. It contains the full CompositeController object (as raw JSON) and the parent object.
type CustomizeRequest[TParent runtime.Object] struct {
	// Controller is the full CompositeController object as received.
	Controller json.RawMessage `json:"controller"`
	// Parent is the parent resource.
	Parent TParent `json:"parent"`
}

// ResourceRule represents a desired related resource as defined by Metacontroller.
type ResourceRule struct {
	// APIVersion is the API version (e.g., "v1" or "apps/v1").
	APIVersion string `json:"apiVersion"`
	// Resource is the canonical, lowercase, plural name of the resource.
	Resource string `json:"resource"`
	// LabelSelector, if set, is a v1.LabelSelector used to select objects.
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`
	// Namespace, if set, restricts selection to a specific namespace.
	Namespace string `json:"namespace,omitempty"`
	// Names is an optional list of individual object names.
	Names []string `json:"names,omitempty"`
}

// CustomizeResponse represents the response from the customize hook.
type CustomizeResponse struct {
	// RelatedResources is a flat list of ResourceRule objects.
	RelatedResources []ResourceRule `json:"relatedResources"`
}

// CustomizeHandler is a function type for processing customize hook requests. It receives a context, the runtime scheme, and a decoded customize request, then returns a customize response or an error.
type CustomizeHandler[TParent runtime.Object] func(ctx context.Context, scheme *runtime.Scheme, req *CustomizeRequest[TParent]) (*CustomizeResponse, error)

// --- HookServer and Functional Options ---

// HookServer is an HTTP server that hosts one or more Metacontroller hook servers.
type HookServer struct {
	addr   string
	scheme *runtime.Scheme
	mux    *http.ServeMux
	server *http.Server
	logger Logger
	debug  bool
}

// Option represents a functional option that configures the HookServer.
type Option func(*HookServer)

// NewHookServer creates a new HookServer that will listen on the provided address
// and use the given Kubernetes scheme for encoding/decoding. The provided options
// register the various hook endpoints.
func NewHookServer(addr string, scheme *runtime.Scheme, opts ...Option) *HookServer {
	hs := &HookServer{
		addr:   addr,
		scheme: scheme,
		mux:    http.NewServeMux(),
		logger: &defaultLogger{},
		debug:  false,
	}
	for _, opt := range opts {
		opt(hs)
	}

	return hs
}

// Logger creates an option that sets a custom logger for the HookServer.
func Logger(logger Logger) Option {
	return func(hs *HookServer) {
		hs.logger = logger
	}
}

// Debug sets the debug flag to true for the HookServer. When debug is true, error responses will include detailed error messages.
func Debug() Option {
	return func(hs *HookServer) {
		hs.debug = true
	}
}

// ListenAndServe starts the HTTP server with the registered endpoints.
func (hs *HookServer) ListenAndServe() error {
	hs.server = &http.Server{
		Addr:    hs.addr,
		Handler: hs.mux,
	}
	hs.logger.Printf("Starting HookServer at %s", hs.addr)

	return hs.server.ListenAndServe()
}

// Shutdown gracefully shuts down the HTTP server using the provided context.
func (hs *HookServer) Shutdown(ctx context.Context) error {
	if hs.server != nil {
		hs.logger.Printf("Shutting down HookServer at %s", hs.addr)
		return hs.server.Shutdown(ctx)
	}

	return nil
}

// SyncHook registers a sync hook handler at the given path.
func SyncHook[TParent runtime.Object](path string, handler SyncHandler[TParent]) Option {
	return func(hs *HookServer) {
		decoder := hs.scheme.Codecs.UniversalDecoder()
		encoder := k8sjson.NewSerializerWithOptions(k8sjson.DefaultMetaFactory, hs.scheme, hs.scheme, k8sjson.SerializerOptions{Yaml: false})
		sh := &syncHookHandler[TParent]{
			scheme:  hs.scheme,
			decoder: decoder,
			encoder: encoder,
			handler: handler,
			logger:  hs.logger,
			debug:   hs.debug,
		}
		hs.mux.Handle(path, sh)
		hs.logger.Printf("Registered sync hook at %q", path)
	}
}

// CustomizeHook registers a customize hook handler at the given path.
func CustomizeHook[TParent runtime.Object](path string, handler CustomizeHandler[TParent]) Option {
	return func(hs *HookServer) {
		decoder := hs.scheme.Codecs.UniversalDecoder()
		ch := &customizeHTTPHandler[TParent]{
			scheme:  hs.scheme,
			decoder: decoder,
			handler: handler,
			logger:  hs.logger,
			debug:   hs.debug,
		}
		hs.mux.Handle(path, ch)
		hs.logger.Printf("Registered customize hook at %q", path)
	}
}

// syncHTTPHandler handles sync hook HTTP requests.
type syncHTTPHandler[TParent runtime.Object] struct {
	scheme  *runtime.Scheme
	decoder runtime.Decoder
	encoder runtime.Encoder
	handler SyncHandler[TParent]
	logger  Logger
	debug   bool
}

// ServeHTTP processes sync hook HTTP requests.
func (sh *syncHTTPHandler[TParent]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var rawReq rawCompositeRequest
	if err := json.NewDecoder(r.Body).Decode(&rawReq); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("SyncHook: error decoding request: %w", err), sh.logger, sh.debug)
		return
	}

	parentObj, _, err := sh.decoder.Decode(rawReq.Parent, nil, nil)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("SyncHook: error decoding parent: %w", err), sh.logger, sh.debug)
		return
	}

	parent, ok := parentObj.(TParent)
	if !ok {
		writeError(w, http.StatusBadRequest, fmt.Errorf("SyncHook: type assertion failure for parent"), sh.logger, sh.debug)
		return
	}

	observedChildren := make(map[schema.GroupVersionKind][]runtime.Object)
	for _, rawList := range rawReq.Children {
		for _, rawChild := range rawList {
			childObj, childGVK, err := sh.decoder.Decode(rawChild, nil, nil)
			if err != nil {
				writeError(w, http.StatusBadRequest, fmt.Errorf("SyncHook: error decoding child: %w", err), sh.logger, sh.debug)
				return
			}
			gvk := *childGVK
			observedChildren[gvk] = append(observedChildren[gvk], childObj)
		}
	}

	req := &CompositeRequest[TParent]{
		Parent:    parent,
		Children:  observedChildren,
		Operation: rawReq.Operation,
	}

	decodedResp, err := sh.handler(r.Context(), sh.scheme, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("SyncHook: handler error: %w", err), sh.logger, sh.debug)
		return
	}

	statusBytes, err := runtime.Encode(sh.encoder, decodedResp.Status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("SyncHook: error encoding status: %w", err), sh.logger, sh.debug)
		return
	}

	desiredChildren := make(map[string][]json.RawMessage)
	for gvk, objs := range decodedResp.Children {
		key := KeyForGVK(gvk)
		var rawList []json.RawMessage
		for _, obj := range objs {
			data, err := runtime.Encode(sh.encoder, obj)
			if err != nil {
				writeError(w, http.StatusInternalServerError, fmt.Errorf("SyncHook: error encoding child: %w", err), sh.logger, sh.debug)
				return
			}
			rawList = append(rawList, json.RawMessage(data))
		}
		desiredChildren[key] = rawList
	}

	rawResp := rawCompositeResponse{
		Status:   statusBytes,
		Children: desiredChildren,
		Patches:  decodedResp.Patches,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(rawResp); err != nil {
		sh.logger.Printf("SyncHook: error encoding response: %v", err)
	}
}

type customizeHTTPHandler[TParent runtime.Object] struct {
	scheme  *runtime.Scheme
	decoder runtime.Decoder
	handler CustomizeHandler[TParent]
	logger  Logger
	debug   bool
}

// ServeHTTP processes customize hook HTTP requests.
func (ch *customizeHTTPHandler[TParent]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var rawReq rawCustomizeRequest
	if err := json.NewDecoder(r.Body).Decode(&rawReq); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("CustomizeHook: error decoding request: %w", err), ch.logger, ch.debug)
		return
	}

	parentObj, _, err := ch.decoder.Decode(rawReq.Parent, nil, nil)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("CustomizeHook: error decoding parent: %w", err), ch.logger, ch.debug)
		return
	}

	parent, ok := parentObj.(TParent)
	if !ok {
		writeError(w, http.StatusBadRequest, fmt.Errorf("CustomizeHook: type assertion failure for parent"), ch.logger, ch.debug)
		return
	}

	customReq := &CustomizeRequest[TParent]{
		Controller: rawReq.Controller,
		Parent:     parent,
	}
	resp, err := ch.handler(r.Context(), ch.scheme, customReq)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("CustomizeHook: handler error: %w", err), ch.logger, ch.debug)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		ch.logger.Printf("CustomizeHook: error encoding response: %v", err)
	}
}
