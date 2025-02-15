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
