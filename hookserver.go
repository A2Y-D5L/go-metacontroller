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

// writeError logs an error and writes an HTTP error response.
// If debug is true, the detailed error message is exposed in the response.
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

// checkMethod verifies that the request uses the expected HTTP method.
// If not, it writes an error response and returns false.
func checkMethod(w http.ResponseWriter, r *http.Request, expectedMethod string, logger Logger, debug bool) bool {
	if r.Method != expectedMethod {
		writeError(w, http.StatusMethodNotAllowed, fmt.Errorf("expected %s method, got %s", expectedMethod, r.Method), logger, debug)
		return false
	}
	return true
}

// decodeJSON decodes JSON from the HTTP request body into the given output.
func decodeJSON(r *http.Request, out interface{}) error {
	return json.NewDecoder(r.Body).Decode(out)
}

// --- Internal types for the sync hook ---

// rawCompositeRequest mirrors the JSON payload for the sync hook.
type rawCompositeRequest struct {
	Parent    json.RawMessage              `json:"parent"`
	Children  map[string][]json.RawMessage `json:"children"`
	Operation string                       `json:"operation"`
}

// rawCompositeResponse is used to encode the sync hook response.
type rawCompositeResponse struct {
	Status   json.RawMessage              `json:"status,omitempty"`
	Children map[string][]json.RawMessage `json:"children,omitempty"`
	Patches  []map[string]interface{}     `json:"patches,omitempty"`
}

// DecodedCompositeRequest represents the fully decoded sync hook request.
type DecodedCompositeRequest[TParent runtime.Object] struct {
	// Parent is the composite (parent) resource.
	Parent TParent
	// Children is a map from GroupVersionKind to slices of decoded child objects.
	Children map[schema.GroupVersionKind][]runtime.Object
	// Operation indicates the type of operation (e.g., "sync" or "finalize").
	Operation string
}

// DecodedCompositeResponse represents the sync hook response.
type DecodedCompositeResponse[TParent runtime.Object] struct {
	// Status is the updated composite (parent) resource.
	Status TParent
	// Children defines the desired state for child objects.
	Children map[schema.GroupVersionKind][]runtime.Object
	// Patches can be used to apply JSON patches to the composite resource.
	Patches []map[string]interface{}
}

// SyncHandler is a function type for processing decoded sync hook requests.
// It receives a context, the runtime scheme, and a decoded composite request,
// then returns a decoded composite response or an error.
type SyncHandler[TParent runtime.Object] func(ctx context.Context, scheme *runtime.Scheme, req *DecodedCompositeRequest[TParent]) (*DecodedCompositeResponse[TParent], error)

// --- Types for the customize hook ---

// CustomizeRequest represents the customize hook request.
// It contains the full CompositeController object (as raw JSON) and the parent object.
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

// CustomizeHandler is a function type for processing customize hook requests.
// It receives a context, the runtime scheme, and a decoded customize request,
// then returns a customize response or an error.
type CustomizeHandler[TParent runtime.Object] func(ctx context.Context, scheme *runtime.Scheme, req *CustomizeRequest[TParent]) (*CustomizeResponse, error)

// --- HookServer and Functional Options ---

// HookServer is a non‑generic server that can host multiple CompositeController hooks.
// Consumers register sync and/or customize handlers via functional options.
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

// WithLogger creates an option that sets a custom logger for the HookServer.
func WithLogger(logger Logger) Option {
	return func(hs *HookServer) {
		hs.logger = logger
	}
}

// WithDebug creates an option that sets the debug flag for the HookServer.
// When debug is true, error responses will include detailed error messages.
func WithDebug(debug bool) Option {
	return func(hs *HookServer) {
		hs.debug = debug
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

// WithSyncHook creates an option that registers a sync hook handler at the given path.
func WithSyncHook[TParent runtime.Object](path string, handler SyncHandler[TParent]) Option {
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

// WithCustomizeHook creates an option that registers a customize hook handler at the given path.
func WithCustomizeHook[TParent runtime.Object](path string, handler CustomizeHandler[TParent]) Option {
	return func(hs *HookServer) {
		decoder := hs.scheme.Codecs.UniversalDecoder()
		ch := &customizeHookHandler[TParent]{
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

// --- Helper: KeyForGVK ---

// KeyForGVK constructs a string key for a given GroupVersionKind in the form "group/version/kind".
// If the group is empty, the key is "version/kind".
func KeyForGVK(gvk schema.GroupVersionKind) string {
	if gvk.Group == "" {
		return fmt.Sprintf("%s/%s", gvk.Version, gvk.Kind)
	}
	return fmt.Sprintf("%s/%s/%s", gvk.Group, gvk.Version, gvk.Kind)
}

// --- syncHookHandler implementation ---

// syncHookHandler handles sync hook HTTP requests.
type syncHookHandler[TParent runtime.Object] struct {
	scheme  *runtime.Scheme
	decoder runtime.Decoder
	encoder runtime.Encoder
	handler SyncHandler[TParent]
	logger  Logger
	debug   bool
}

// ServeHTTP processes sync hook HTTP requests.
func (sh *syncHookHandler[TParent]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !checkMethod(w, r, http.MethodPost, sh.logger, sh.debug) {
		return
	}

	var rawReq rawCompositeRequest
	if err := decodeJSON(r, &rawReq); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("SyncHook: error decoding request: %w", err), sh.logger, sh.debug)
		return
	}

	// Decode parent.
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

	// Decode children and group by GVK.
	childrenMap := make(map[schema.GroupVersionKind][]runtime.Object)
	for _, rawList := range rawReq.Children {
		for _, rawChild := range rawList {
			childObj, childGVK, err := sh.decoder.Decode(rawChild, nil, nil)
			if err != nil {
				writeError(w, http.StatusBadRequest, fmt.Errorf("SyncHook: error decoding child: %w", err), sh.logger, sh.debug)
				return
			}
			gvk := *childGVK
			childrenMap[gvk] = append(childrenMap[gvk], childObj)
		}
	}

	decodedReq := &DecodedCompositeRequest[TParent]{
		Parent:    parent,
		Children:  childrenMap,
		Operation: rawReq.Operation,
	}

	decodedResp, err := sh.handler(r.Context(), sh.scheme, decodedReq)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("SyncHook: handler error: %w", err), sh.logger, sh.debug)
		return
	}

	// Encode updated parent.
	statusBytes, err := runtime.Encode(sh.encoder, decodedResp.Status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("SyncHook: error encoding status: %w", err), sh.logger, sh.debug)
		return
	}

	// Encode children.
	encodedChildren := make(map[string][]json.RawMessage)
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
		encodedChildren[key] = rawList
	}

	rawResp := rawCompositeResponse{
		Status:   statusBytes,
		Children: encodedChildren,
		Patches:  decodedResp.Patches,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(rawResp); err != nil {
		sh.logger.Printf("SyncHook: error encoding response: %v", err)
	}
}

// --- customizeHookHandler implementation ---

// customizeHookHandler handles customize hook HTTP requests.
type customizeHookHandler[TParent runtime.Object] struct {
	scheme  *runtime.Scheme
	decoder runtime.Decoder
	handler CustomizeHandler[TParent]
	logger  Logger
	debug   bool
}

// ServeHTTP processes customize hook HTTP requests.
func (ch *customizeHookHandler[TParent]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !checkMethod(w, r, http.MethodPost, ch.logger, ch.debug) {
		return
	}
	var rawReq struct {
		Controller json.RawMessage `json:"controller"`
		Parent     json.RawMessage `json:"parent"`
	}
	if err := decodeJSON(r, &rawReq); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("CustomizeHook: error decoding request: %w", err), ch.logger, ch.debug)
		return
	}
	// Decode parent.
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
