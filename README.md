# go-metacontroller

A framework for building [Metacontroller](https://metacontroller.github.io/metacontroller/) webhook servers in Go.

## Features

- **Flexible Hook Registration:** Easily register multiple sync and customize hooks for different parent resource types.
- **Customizable Logging:** Use the default logger or provide your own implementation.
- **Kubernetes API Integration:** Seamlessly decode and encode Kubernetes API objects using a provided runtime scheme.

## Installation

```bash
go get github.com/a2y-d5l/go-metacontroller
```

## Usage

The following demonstrates how to create a `HookServer`, register `sync` and `customize` hooks, and start the server.

```go
package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/a2y-d5l/go-metacontroller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Parent represents a custom parent resource.
type Parent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              map[string]any `json:"spec,omitempty"`
	Status            map[string]any `json:"status,omitempty"`
}

// DeepCopyObject implements the runtime.Object interface.
func (p *Parent) DeepCopyObject() runtime.Object {
	// Deep copy implementation (omitted for brevity)
	return p
}

// SyncHandler processes sync hook requests.
func SyncHandler(ctx context.Context, scheme *runtime.Scheme, req *metacontroller.CompositeRequest[*Parent]) (*metacontroller.CompositeResponse[*Parent], error) {
	// Implement your sync logic here.
	// For example, update the status and desired child resources.
	resp := &metacontroller.CompositeResponse[*MyParent]{
		Status: req.Parent,
		Children: map[schema.GroupVersionKind][]runtime.Object{
			{Group: "apps", Version: "v1", Kind: "Deployment"}: {},
		},
	}
	return resp, nil
}

// CustomizeHandler processes customize hook requests.
func CustomizeHandler(ctx context.Context, scheme *runtime.Scheme, req *metacontroller.CustomizeRequest[*Parent]) (*metacontroller.CustomizeResponse, error) {
	// Define related resources based on the parent resource.
	resp := &metacontroller.CustomizeResponse{
		RelatedResources: []metacontroller.ResourceRule{
			{
				APIVersion: "apps/v1",
				Resource:   "deployments",
			},
		},
	}
	return resp, nil
}

func main() {
	// Create a new Kubernetes runtime scheme and register your types.
	scheme := runtime.NewScheme()
	// (Register Parent and any other types with the scheme as needed)

	// Create a new HookServer with sync and customize hooks.
	hs := metacontroller.NewHookServer(":8080", scheme,
		metacontroller.SyncHook("/sync/parents", SyncHandler),
		metacontroller.CustomizeHook("/customize/parents", CustomizeHandler),
		metacontroller.Debug(), // Enable debug mode for detailed errors.
	)

	// Start the server in a separate goroutine.
	go func() {
		log.Printf("HookServer starting on %s", hs.Addr)
		if err := hs.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HookServer error: %v", err)
		}
	}()

	// Simulate running server for a duration.
	time.Sleep(10 * time.Second)

	// Gracefully shut down the server.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := hs.Shutdown(ctx); err != nil {
		log.Fatalf("Error shutting down HookServer: %v", err)
	}
	log.Println("HookServer graceful shutdown complete.")
}
```

## Handler Types

### Sync Handler

**Type:** `SyncHandler[TParent runtime.Object]`

Processes sync hook requests to update the parent resource status and define desired child objects.

**Parameters:**

- `ctx`: The request context.
- `scheme`: The Kubernetes runtime scheme for encoding/decoding.
- `req`: A `CompositeRequest` containing:
  - `Parent`: The composite (parent) resource.
  - `Children`: A map grouping child objects by their `GroupVersionKind`.
  - `Operation`: The operation type (e.g., `sync` or `finalize`).

**Returns:** A `CompositeResponse` with the updated parent status, desired child resources.

### Customize Handler

**Type:** `CustomizeHandler[TParent runtime.Object]`

Processes customize hook requests to define related resources for the parent resource.

**Parameters:**

- `ctx`: The request context.
- `scheme`: The Kubernetes runtime scheme.
- `req`: A CustomizeRequest containing:
  - `Controller`: The raw JSON of the full CompositeController object.
  - `Parent`: The parent resource.

**Returns:** A `CustomizeResponse` that includes a list of ResourceRule objects specifying related resources.

## API Overview

### `HookServer`

The core server that handles HTTP requests for registered hooks. It uses an internal HTTP multiplexer and supports graceful shutdown.

### Functional Options

Configure the `HookServer`.

- `Logger(Logger)`: Set a custom logger.
- `Debug()`: Enables debug mode.
- `SyncHook(path string, handler SyncHandler[TParent])`: Register a `sync` hook handler to handle requests at the specified HTTP path.
- `CustomizeHook(path string, handler CustomizeHandler[TParent])`: Register a `customize` hook handler to handle requests at the specified HTTP path.

### Helper Functions:

- `KeyForGVK(gvk schema.GroupVersionKind) string`: Constructs a string key for a given GroupVersionKind in the format group/version/kind (or version/kind if the group is empty).

For more detailed API usage, refer to the source code documentation.

## Contributing

Contributions are welcome! If you find a bug or have a feature request, please open an issue or submit a pull request on GitHub.

## License

This project is licensed under the [Apache 2.0 License](./LICENSE).

## Acknowledgements

This library builds on the amazing work done by the [Metacontroller authors](https://github.com/metacontroller/metacontroller/graphs/contributors) and leverages Kubernetes API machinery (from k8s.io/apimachinery) for robust encoding and decoding of runtime objects. Many thanks to the Kubernetes community for their invaluable contributions.
