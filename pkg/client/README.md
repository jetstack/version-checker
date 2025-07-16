# pkg/client

This package provides specialized clients for interacting with different container registries or artifact repositories, particularly where custom authentication and authorization mechanisms are required.

## Philosophy

Clients in this package are **only created where specific authentication or authorization is required**. For common use cases, the default client should be sufficient. This design helps minimize overhead and complexity for common cases while still allowing flexibility for secure or proprietary registries.

## Standard Client: `OCI`

The `OCI` client is the **standard client** and acts as the baseline fallback for all operations. All other clients should defer to `OCI` where possible to ensure consistent behavior and compatibility across registry interactions.

## Custom Clients

Custom clients are provided to support more advanced workflows, particularly where the registry:

* Requires custom token exchange or API authentication
* Provides richer metadata or querying capabilities

These clients often rely on internal APIs to **provide a more efficient way of interrogating versions/tags**, improving performance and accuracy over standard OCI calls.

## Guidelines

* Use the `OCI` client unless there is a clear need for custom auth or advanced metadata.
* Custom clients should implement or extend common interfaces to preserve interchangeability.
* Prefer internal APIs only where they offer real advantages (e.g., performance, data quality).

## Future Direction

All new client implementations should:

* Fall back to `OCI` when internal or authenticated APIs are not available.
* Expose a consistent interface to allow consumers to switch between clients with minimal changes.

---

This package is intended for internal use, but may be extended in the future to support user-defined plugins or external registry integrations.
