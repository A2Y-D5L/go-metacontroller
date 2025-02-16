# go-metacontroller codebase

## Tree View

    .
    ├─composite
    │ ├─customize.go
    │ ├─finalize.go
    │ └─sync.go
    ├─go.mod
    ├─hookserver.go
    └─http_handlers.go

## Content

### composite/customize.go

```go
package composite

import (
	"context"
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Request represents the customize hook request. It contains the full CompositeController object (as raw JSON) and the parent object.
type CustomizeRequest[P client.Object] struct {
	// Controller is the full CompositeController object as received.
	Controller json.RawMessage `json:"controller"`
	// Parent is the parent resource.
	Parent P `json:"parent"`
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
type CustomizeHandler[P client.Object] func(context.Context, *runtime.Scheme, *CustomizeRequest[P]) (*CustomizeResponse, error)

```

### composite/finalize.go
```go
package composite

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	api "k8s.io/apimachinery/pkg/runtime"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// FinalizeRequest represents the fully decoded finalize hook request.
type FinalizeRequest[P client.Object] struct {
	// Parent is the composite (parent) resource.
	Parent P
	// Children is a map from GroupVersionKind to slices of decoded child objects.
	Children map[schema.GroupVersionKind][]client.Object
}

// FinalizeResponse represents the finalize hook response.
type FinalizeResponse[TParent client.Object] struct {
	// Status is the updated composite (parent) resource.
	Status TParent
	// Children defines the desired state for child objects.
	Children map[schema.GroupVersionKind][]client.Object
	// Finalized indicates whether the parent resource should be marked as finalized.
	Finalized bool
}

// FinalizeHandler is a function type for processing finalize requests.
type FinalizeHandler[P client.Object] func(ctx context.Context, scheme *api.Scheme, req *FinalizeRequest[P]) (*FinalizeResponse[P], error)

```

### composite/sync.go
```go
package composite

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	api "k8s.io/apimachinery/pkg/runtime"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// SyncRequest represents the fully decoded sync hook request.
type SyncRequest[P client.Object] struct {
	// Parent is the composite (parent) resource.
	Parent P
	// Children is a map from GroupVersionKind to slices of decoded child objects.
	Children map[schema.GroupVersionKind][]client.Object
	// Finalizing indicates the type of sync operation (sync=false, finalize=true).
	Finalizing bool
}

// CompositeResponse represents the sync hook response.
type CompositeResponse[TParent client.Object] struct {
	// Status is the updated composite (parent) resource.
	Status TParent
	// Children defines the desired state for child objects.
	Children map[schema.GroupVersionKind][]client.Object
	// Finalized indicates whether the parent resource should be marked as finalized.
	Finalized bool
}

// SyncHandler is a function type for processing decoded sync hook requests. It receives a context, the runtime scheme, and a decoded composite request, then returns a decoded composite response or an error.
type SyncHandler[TParent client.Object] func(ctx context.Context, scheme *api.Scheme, req *SyncRequest[TParent]) (*CompositeResponse[TParent], error)

```

### go.mod
```mod
module github.com/a2y-d5l/go-metacontroller

go 1.23.0

toolchain go1.23.4

require (
	k8s.io/apimachinery v0.32.1
	sigs.k8s.io/controller-runtime v0.20.2
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/emicklei/go-restful/v3 v3.11.0 // indirect
	github.com/evanphx/json-patch/v5 v5.9.11 // indirect
	github.com/fxamacker/cbor/v2 v2.7.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/gnostic-models v0.6.8 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	golang.org/x/net v0.30.0 // indirect
	golang.org/x/oauth2 v0.23.0 // indirect
	golang.org/x/sys v0.26.0 // indirect
	golang.org/x/term v0.25.0 // indirect
	golang.org/x/text v0.19.0 // indirect
	golang.org/x/time v0.7.0 // indirect
	google.golang.org/protobuf v1.35.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/api v0.32.1 // indirect
	k8s.io/client-go v0.32.1 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	k8s.io/kube-openapi v0.0.0-20241105132330-32ad38e42d3f // indirect
	k8s.io/utils v0.0.0-20241104100929-3ea5e8cea738 // indirect
	sigs.k8s.io/json v0.0.0-20241010143419-9aa6b5e7a4b3 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.2 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)

```

### hookserver.go
```go
// Package metacontroller provides a webhook server framework for implementing
// CompositeController hooks for Metacontroller. Consumers can run multiple hooks
// (for various parent resource types) by supplying sync and/or customize handlers
// via functional options. The HookServer creates its own HTTP multiplexer so that it
// isn’t bound to the default HTTP server.
package metacontroller

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/a2y-d5l/go-metacontroller/controller/composite"
	"github.com/a2y-d5l/go-metacontroller/controller/customize"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// HookServer is an HTTP server that hosts one or more Metacontroller hook servers.
type HookServer struct {
	addr   string
	scheme *runtime.Scheme
	mux    *http.ServeMux
	server *http.Server
	logger *slog.Logger
	debug  bool
}

// NewHookServer creates a new HookServer that will listen on the provided address
// and use the given Kubernetes scheme for encoding/decoding. The provided options
// register the various hook endpoints.
func NewHookServer(scheme *runtime.Scheme, opts ...Option) *HookServer {
	hs := &HookServer{
		addr:   ":8080",
		scheme: scheme,
		mux:    http.NewServeMux(),
		logger: slog.Default(),
	}
	for _, opt := range opts {
		opt(hs)
	}

	return hs
}

// Option represents a functional option that configures the HookServer.
type Option func(*HookServer)

// Addr sets the address for the HookServer. (Default: ":8080")
func Addr(addr string) Option {
	return func(hs *HookServer) {
		hs.addr = addr
	}
}

// Logger creates an option that sets a custom logger for the HookServer. (Default: slog.Default())
func Logger(logger *slog.Logger) Option {
	return func(hs *HookServer) {
		hs.logger = logger
	}
}

// CompositeHook is a functional option that registers a CompositeController hook with the HookServer.
type CompositeHook Option

func CompositeController(hooks ...CompositeHook) Option {
	return func(hs *HookServer) {
		for _, hook := range hooks {
			hook(hs)
		}
	}
}

func SyncHook[P client.Object](gvr schema.GroupVersionResource, handler composite.SyncHandler[P]) CompositeHook {
	return CompositeHook(func(hs *HookServer) {
		resource := fmt.Sprintf("%s/%s", gvr.GroupResource().String(), gvr.Version)
		path := "/hooks/sync/" + resource
		hs.mux.Handle("POST "+path, &syncHandler[P]{
			scheme:  hs.scheme,
			decoder: hs.scheme.Codecs.UniversalDecoder(),
			encoder: hs.scheme.Codecs.LegacyCodec(),
			handler: handler,
			logger:  hs.logger,
			debug:   hs.debug,
		})
		hs.logger.Info("Registered sync hook at %q for %q", path, gvr.String())
	})
}

func FinalizeHook[P client.Object](gvr schema.GroupVersionResource, handler composite.SyncHandler[P]) CompositeHook {
	return CompositeHook(func(hs *HookServer) {
		resource := fmt.Sprintf("%s/%s", gvr.GroupResource().String(), gvr.Version)
		path := "/hooks/finalize/" + resource
		hs.mux.Handle("POST "+path, &syncHandler[P]{
			scheme:  hs.scheme,
			decoder: hs.scheme.Codecs.UniversalDecoder(),
			encoder: hs.scheme.Codecs.LegacyCodec(),
			handler: handler,
			logger:  hs.logger,
		})
		hs.logger.Info("Registered finalize hook at %q for %q", path, gvr.String())
	})
}

func CustomizeHook[P client.Object](gvr schema.GroupVersionResource, handler customize.Handler[P]) CompositeHook {
	return CompositeHook(func(hs *HookServer) {
		resource := fmt.Sprintf("%s/%s", gvr.GroupResource().String(), gvr.Version)
		path := "/hooks/customize/" + resource
		hs.mux.Handle("POST "+path, &customizeHTTPHandler[P]{
			scheme:  hs.scheme,
			decoder: hs.scheme.Codecs.UniversalDecoder(),
			handler: handler,
			logger:  hs.logger,
			debug:   hs.debug,
		})
		hs.logger.Info("Registered customize hook at %q for %q", path, gvr.String())
	})
}

// ListenAndServe starts the HTTP server with the registered endpoints.
func (hs *HookServer) ListenAndServe() error {
	hs.server = &http.Server{
		Addr:    hs.addr,
		Handler: hs.mux,
	}
	hs.logger.Info("Starting HookServer at %s", hs.addr)

	return hs.server.ListenAndServe()
}

// Shutdown gracefully shuts down the HTTP server using the provided context.
func (hs *HookServer) Shutdown(ctx context.Context) error {
	if hs.server != nil {
		hs.logger.Info("Shutting down HookServer at %s", hs.addr)
		return hs.server.Shutdown(ctx)
	}

	return nil
}

```

### http_handlers.go
```go
package metacontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/a2y-d5l/go-metacontroller/controller/composite"
	"github.com/a2y-d5l/go-metacontroller/controller/customize"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type (
	// rawCompositeRequest mirrors the JSON payload for the sync hook.
	rawCompositeRequest struct {
		Parent     json.RawMessage              `json:"parent"`
		Children   map[string][]json.RawMessage `json:"children"`
		Finalizing bool                         `json:"finalizing"`
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
	handler composite.SyncHandler[P]
	logger  *slog.Logger
	debug   bool
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
		writeError(r.Context(), w, http.StatusBadRequest, fmt.Errorf("SyncHook: type assertion failure for parent"), sh.logger)

		return
	}

	observedChildren := make(map[schema.GroupVersionKind][]client.Object)
	for _, rawList := range rawReq.Children {
		for _, rawChild := range rawList {
			childObj, childGVK, err := sh.decoder.Decode(rawChild, nil, nil)
			if err != nil {
				sh.logger.ErrorContext(r.Context(), "SyncHook: error decoding child: "+err.Error(), slog.String("child", string(rawChild)))

				continue
			}

			child, ok := childObj.(client.Object)
			if !ok {
				sh.logger.ErrorContext(r.Context(), "SyncHook: type assertion failure for child", slog.String("child", string(rawChild)))

				continue
			}
			observedChildren[*childGVK] = append(observedChildren[*childGVK], child)
		}
	}

	resp, err := sh.handler(r.Context(), sh.scheme, &composite.SyncRequest[P]{
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
		Status:    statusBytes,
		Children:  desiredChildren,
		Finalized: resp.Finalized,
	}); err != nil {
		sh.logger.ErrorContext(r.Context(), "SyncHook: error encoding response: "+err.Error())
	}
}

type customizeHTTPHandler[P client.Object] struct {
	scheme  *runtime.Scheme
	decoder runtime.Decoder
	handler customize.Handler[P]
	logger  *slog.Logger
	debug   bool
}

// ServeHTTP processes customize hook HTTP requests.
func (ch *customizeHTTPHandler[P]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	resp, err := ch.handler(r.Context(), ch.scheme, &customize.Request[P]{
		Controller: rawReq.Controller,
		Parent:     parent,
	})
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, fmt.Errorf("CustomizeHook: handler error: %w", err), ch.logger)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		ch.logger.Error("CustomizeHook: error encoding response: " + err.Error())
	}
}

```
