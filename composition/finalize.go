package composition

import (
	"context"

	api "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
type FinalizeResponse[P client.Object] struct {
	// Status is the updated composite (parent) resource.
	Status P
	// Children defines the desired state for child objects.
	Children map[schema.GroupVersionKind][]client.Object
	// Finalized indicates whether the parent resource should be marked as finalized.
	Finalized bool
}

// Finalizer is an interface for processing finalize requests.
type Finalizer[P client.Object] interface {
	// Finalize is a function that processes finalize requests.
	// It receives a context, the runtime scheme, and a decoded finalize request,
	// then returns a finalize response or an error.
	Finalize(
		ctx context.Context,
		scheme *api.Scheme,
		req *FinalizeRequest[P],
	) (*FinalizeResponse[P], error)
}

// FinalizerFunc is a functional implementation of the Finalizer interface.
type FinalizeFunc[P client.Object] func(
	ctx context.Context,
	scheme *api.Scheme,
	req *FinalizeRequest[P],
) (*FinalizeResponse[P], error)

// Finalize implements the Finalizer interface.
func (fn FinalizeFunc[P]) Finalize(ctx context.Context, scheme *api.Scheme, req *FinalizeRequest[P]) (*FinalizeResponse[P], error) {
	return fn(ctx, scheme, req)
}
