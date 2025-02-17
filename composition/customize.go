package composition

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

// Customizer is an interface for processing customize hook requests.
type Customizer[P client.Object] interface {
	// Customize is a function that processes customize requests. It receives a context, the runtime scheme, and a decoded customize request, then returns a customize response or an error.
	Customize(
		ctx context.Context,
		scheme *runtime.Scheme,
		req *CustomizeRequest[P],
	) (*CustomizeResponse, error)
}

// CustomizerFunc is a functional implementation of the Customizer interface.
type CustomizeFunc[P client.Object] func(
	ctx context.Context,
	scheme *runtime.Scheme,
	req *CustomizeRequest[P],
) (*CustomizeResponse, error)

// Customize implements the Customizer interface.
func (fn CustomizeFunc[P]) Customize(ctx context.Context, scheme *runtime.Scheme, req *CustomizeRequest[P]) (*CustomizeResponse, error) {
	return fn(ctx, scheme, req)
}
