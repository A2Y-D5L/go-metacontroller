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
