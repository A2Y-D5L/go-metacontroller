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

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/a2y-d5l/go-metacontroller/composition"
)

// HookServer is an HTTP server that hosts one or more Metacontroller hook servers.
type HookServer struct {
	addr   string
	scheme *runtime.Scheme
	codecs serializer.CodecFactory
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
	hs.codecs = serializer.NewCodecFactory(scheme)
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

func SyncHook[P client.Object](gvr schema.GroupVersionResource, syncer composition.Syncer[P]) CompositeHook {
	return CompositeHook(func(hs *HookServer) {
		resource := fmt.Sprintf("%s/%s", gvr.GroupResource().String(), gvr.Version)
		path := "/hooks/sync/" + resource
		hs.mux.Handle("POST "+path, &syncHandler[P]{
			scheme:  hs.scheme,
			decoder: hs.codecs.UniversalDecoder(),
			encoder: hs.codecs.LegacyCodec(gvr.GroupVersion()),
			syncer:  syncer,
			logger:  hs.logger,
		})
		hs.logger.Info("Registered sync hook at %q for %q", path, gvr.String())
	})
}

func FinalizeHook[P client.Object](gvr schema.GroupVersionResource, finalizer composition.Finalizer[P]) CompositeHook {
	return CompositeHook(func(hs *HookServer) {
		resource := fmt.Sprintf("%s/%s", gvr.GroupResource().String(), gvr.Version)
		path := "/hooks/finalize/" + resource
		hs.mux.Handle("POST "+path, &finalizeHandler[P]{
			scheme:    hs.scheme,
			decoder:   hs.codecs.UniversalDecoder(),
			finalizer: finalizer,
			logger:    hs.logger,
		})
		hs.logger.Info("Registered finalize hook at %q for %q", path, gvr.String())
	})
}

func CustomizeHook[P client.Object](gvr schema.GroupVersionResource, customizer composition.Customizer[P]) CompositeHook {
	return CompositeHook(func(hs *HookServer) {
		resource := fmt.Sprintf("%s/%s", gvr.GroupResource().String(), gvr.Version)
		path := "/hooks/customize/" + resource
		hs.mux.Handle("POST "+path, &customizeHandler[P]{
			scheme:     hs.scheme,
			decoder:    hs.codecs.UniversalDecoder(),
			customizer: customizer,
			logger:     hs.logger,
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
