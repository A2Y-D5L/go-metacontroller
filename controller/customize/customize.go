package customize

import (
	"context"
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Request represents the customize hook request. It contains the full CompositeController object (as raw JSON) and the parent object.
type Request[P client.Object] struct {
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
type Response struct {
	// RelatedResources is a flat list of ResourceRule objects.
	RelatedResources []ResourceRule `json:"relatedResources"`
}

// Handler is a function type for processing customize hook requests. It receives a context, the runtime scheme, and a decoded customize request, then returns a customize response or an error.
type Handler[P client.Object] func(context.Context, *runtime.Scheme, *Request[P]) (*Response, error)
