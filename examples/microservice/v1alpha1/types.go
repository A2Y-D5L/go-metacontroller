package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MicroserviceSpec defines the desired state of Microservice.
type MicroserviceSpec struct {
	// Image is the container image to deploy.
	// +kubebuilder:validation:Required
	Image string `json:"image"`

	// Replicas is the number of pod replicas to run.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// Port is the HTTP port for the container and Service.
	// +kubebuilder:validation:Required
	Port int32 `json:"port"`

	// Exposure determines how the service is exposed.
	// It can be one of: "cluster", "private", or "public".
	// +kubebuilder:validation:Enum=cluster;private;public
	// +kubebuilder:default=cluster
	// +optional
	Exposure string `json:"exposure,omitempty"`
}

// MicroserviceStatus defines the observed state of Microservice.
type MicroserviceStatus struct {
	// Conditions represents the latest available observations of the object's state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Image",type="string",JSONPath=".spec.image"
//+kubebuilder:printcolumn:name="Replicas",type="integer",JSONPath=".spec.replicas"
//+kubebuilder:printcolumn:name="Port",type="integer",JSONPath=".spec.port"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Microservice is the Schema for the microservices API.
type Microservice struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MicroserviceSpec   `json:"spec,omitempty"`
	Status MicroserviceStatus `json:"status,omitempty"`
}

// Ensure Microservice implements client.Object.
var _ client.Object = &Microservice{}

// DeepCopyInto copies the receiver, writing into out. in must be non-nil.
func (in *Microservice) DeepCopyInto(out *Microservice) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	if in.Status.Conditions != nil {
		in, out := &in.Status.Conditions, &out.Status.Conditions
		*out = make([]metav1.Condition, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy creates a new Microservice by deep copying the receiver.
func (in *Microservice) DeepCopy() *Microservice {
	if in == nil {
		return nil
	}
	out := new(Microservice)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject implements the runtime.Object interface.
func (in *Microservice) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

//+kubebuilder:object:root=true

// MicroserviceList contains a list of Microservice.
type MicroserviceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Microservice `json:"items"`
}

// Ensure MicroserviceList implements client.ObjectList.
var _ client.ObjectList = &MicroserviceList{}

// DeepCopyInto copies the receiver, writing into out. in must be non-nil.
func (in *MicroserviceList) DeepCopyInto(out *MicroserviceList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Microservice, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy creates a new MicroserviceList by deep copying the receiver.
func (in *MicroserviceList) DeepCopy() *MicroserviceList {
	if in == nil {
		return nil
	}
	out := new(MicroserviceList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject implements the runtime.Object interface.
func (in *MicroserviceList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
