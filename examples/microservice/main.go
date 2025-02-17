// Package main demonstrates a real-world composite controller that reconciles
// a Microservice custom resource into a Deployment and a Service.
package main

import (
	"context"
	"log"
	"net/http"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/a2y-d5l/go-metacontroller"
	"github.com/a2y-d5l/go-metacontroller/composition"
	"github.com/a2y-d5l/go-metacontroller/examples/microservice/v1alpha1"
)

// sync reads a Microservice spec to create a Deployment and a Service.
func sync(ctx context.Context, scheme *runtime.Scheme, req *composition.SyncRequest[*v1alpha1.Microservice]) (*composition.SyncResponse[*v1alpha1.Microservice], error) {
	name := req.Parent.GetName()
	namespace := req.Parent.GetNamespace()
	if namespace == "" {
		namespace = "default"
	}

	// Create a Deployment for the microservice.
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-deploy",
			Namespace: namespace,
			Labels: map[string]string{
				"app": name,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &req.Parent.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "microservice",
							Image: req.Parent.Spec.Image,
							Ports: []corev1.ContainerPort{
								{ContainerPort: req.Parent.Spec.Port},
							},
						},
					},
				},
			},
		},
	}

	// Determine the Service type based on exposure.
	svcType := corev1.ServiceTypeClusterIP
	if req.Parent.Spec.Exposure == "public" {
		svcType = corev1.ServiceTypeLoadBalancer
	}

	// Create a Service to expose the microservice.
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-svc",
			Namespace: namespace,
			Labels:    map[string]string{"app": name},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": name},
			Ports: []corev1.ServicePort{
				{
					Port:       req.Parent.Spec.Port,
					TargetPort: intstr.FromInt(int(req.Parent.Spec.Port)),
				},
			},
			Type: svcType,
		},
	}

	// Return the result of the sync operation.
	// In this example, we do not update the parent status.
	return &composition.SyncResponse[*v1alpha1.Microservice]{
		Status:   req.Parent,
		Children: []client.Object{deployment, service},
	}, nil
}

func main() {
	// Create a new runtime scheme.
	scheme := runtime.NewScheme()

	// Register API types for the Microservice children.
	if err := appsv1.AddToScheme(scheme); err != nil {
		log.Fatalf("Failed to add appsv1 to scheme: %v", err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		log.Fatalf("Failed to add corev1 to scheme: %v", err)
	}

	// Create a HookServer with our sync hook registered.
	hs := metacontroller.NewHookServer(scheme,
		metacontroller.CompositeController(
			metacontroller.SyncHook[*v1alpha1.Microservice](
				v1alpha1.MicroserviceGroupVersionResource,
				composition.SyncerFunc[*v1alpha1.Microservice](sync),
			),
		),
	)

	// Start the HookServer.
	if err := hs.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("HookServer error: %v", err)
	}
}
