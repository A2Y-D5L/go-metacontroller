package composite

import (
	"context"

	api "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

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

// SyncResponse represents the sync hook response.
type SyncResponse[P client.Object] struct {
	// Status is the updated composite (parent) resource.
	Status P
	// Children defines the desired state for child objects.
	Children map[schema.GroupVersionKind][]client.Object
	// Finalized indicates whether the parent resource should be marked as finalized.
}

// Syncer is an interface for processing sync hook requests.
type Syncer[P client.Object] interface {
	// Sync is a function that processes sync requests.
	// It receives a context, the runtime scheme, and a decoded sync request,
	// then returns a sync response or an error.
	Sync(
		ctx context.Context,
		scheme *api.Scheme,
		req *SyncRequest[P],
	) (*SyncResponse[P], error)
}

// SyncerFunc is a functional implementation of the Syncer interface.
type SyncerFunc[P client.Object] func(
	ctx context.Context,
	scheme *api.Scheme,
	req *SyncRequest[P],
) (*SyncResponse[P], error)

// Sync implements the Syncer interface.
func (fn SyncerFunc[P]) Sync(ctx context.Context, scheme *api.Scheme, req *SyncRequest[P]) (*SyncResponse[P], error) {
	return fn(ctx, scheme, req)
}
