# go-metacontroller

go-metacontroller is a framework for building Metacontroller webhook servers in Go.

## Features

- **Flexible Hook Registration:** Easily register multiple sync and customize hooks for different parent resource types.
- **Customizable Logging:** Use the default logger or provide your own implementation.
- **Debug Mode:** Enable detailed error responses to aid development and troubleshooting.
- **Kubernetes API Integration:** Seamlessly decode and encode Kubernetes API objects using a provided runtime scheme.

## Installation

Install the library using `go get`:

```bash
go get github.com/A2Y-D5L/go-metacontroller
```

## Usage

### Creating a Hook Server

The following example demonstrates how to create a `HookServer`, register `sync` and `customize` hooks, and start the server.

```go
package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/yourusername/metacontroller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// MyParent represents a custom parent resource.
type MyParent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              map[string]interface{} `json:"spec,omitempty"`
	Status            map[string]interface{} `json:"status,omitempty"`
}

// DeepCopyObject implements the runtime.Object interface.
func (m *MyParent) DeepCopyObject() runtime.Object {
	// Deep copy implementation (omitted for brevity)
	return m
}

// MySyncHandler processes sync hook requests.
func MySyncHandler(ctx context.Context, scheme *runtime.Scheme, req *metacontroller.DecodedCompositeRequest[*MyParent]) (*metacontroller.DecodedCompositeResponse[*MyParent], error) {
	// Implement your sync logic here.
	// For example, update the status and desired child resources.
	resp := &metacontroller.DecodedCompositeResponse[*MyParent]{
		Status: req.Parent,
		Children: map[schema.GroupVersionKind][]runtime.Object{
			{Group: "apps", Version: "v1", Kind: "Deployment"}: {},
		},
		Patches: nil, // Optional JSON patches can be provided here.
	}
	return resp, nil
}

// MyCustomizeHandler processes customize hook requests.
func MyCustomizeHandler(ctx context.Context, scheme *runtime.Scheme, req *metacontroller.CustomizeRequest[*MyParent]) (*metacontroller.CustomizeResponse, error) {
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
	// (Register MyParent and any other types with the scheme as needed)

	// Create a new HookServer with sync and customize hooks.
	hs := metacontroller.NewHookServer(":8080", scheme,
		metacontroller.WithSyncHook("/sync", MySyncHandler),
		metacontroller.WithCustomizeHook("/customize", MyCustomizeHandler),
		metacontroller.WithDebug(true), // Enable debug mode for detailed errors.
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
		log.Fatalf("Error shutting down server: %v", err)
	}
	log.Println("HookServer gracefully stopped")
}
```

## Handler Types

### Sync Handler

Type: `SyncHandler[TParent runtime.Object]`

Processes sync hook requests to update the parent resource status and define desired child objects.

Parameters:

- ctx: The request context.
- scheme: The Kubernetes runtime scheme for encoding/decoding.
- req: A `CompositeRequest` containing:
  - Parent: The composite (parent) resource.
  - Children: A map grouping child objects by their `GroupVersionKind`.
  - Operation: The operation type (e.g., `sync` or `finalize`).

Returns: A `CompositeResponse` with the updated parent status, desired child resources.

### Customize Handler

**Type:** `CustomizeHandler[TParent runtime.Object]`

Processes customize hook requests to define related resources for the parent resource.

**Parameters:**

- `ctx`: The request context.
- `scheme`: The Kubernetes runtime scheme.
- `req`: A CustomizeRequest containing:
  - Controller: The raw JSON of the full CompositeController object.
  - Parent: The parent resource.

**Returns:** A `CustomizeResponse` that includes a list of ResourceRule objects specifying related resources.

## API Overview

### `HookServer`

The core server that handles HTTP requests for registered hooks. It uses an internal HTTP multiplexer and supports graceful shutdown.

### Functional Options

Configure the `HookServer`.

- `WithLogger(Logger)`: Set a custom logger.
- `WithDebug(bool)`: Enable or disable debug mode.
- `WithSyncHook(path string, handler SyncHandler[TParent])`: Register a sync hook handler at the specified HTTP path.
- `WithCustomizeHook(path string, handler CustomizeHandler[TParent])`: Register a customize hook handler at the specified HTTP path.

### Helper Functions:

- `KeyForGVK(gvk schema.GroupVersionKind) string`: Constructs a string key for a given GroupVersionKind in the format group/version/kind (or version/kind if the group is empty).

For more detailed API usage, refer to the source code documentation.

## Contributing

Contributions are welcome! If you find a bug or have a feature request, please open an issue or submit a pull request on GitHub.

## License

This project is licensed under the [Apache 2.0 License](./LICENSE).

## Acknowledgements

This library builds on the amazing work done by the Metacontroller authors and leverages Kubernetes API machinery (from k8s.io/apimachinery) for robust encoding and decoding of runtime objects. Many thanks to the Kubernetes community for their invaluable contributions.
