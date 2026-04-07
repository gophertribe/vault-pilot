# ADR: Internal Plugins Over External Binaries

## Status

Accepted (2024-02)

## Context

We need a plugin architecture for Vault Pilot to support multiple domain features (email, GTD, business, software) while keeping the codebase maintainable.

Go offers two main approaches:
1. **External plugins**: Runtime-loaded `.so` files via the `plugin` package
2. **Internal plugins**: Go packages compiled into a single binary

## Decision

We will use **internal plugins** (Go packages compiled into one binary).

## Rationale

### Problems with External Plugins

1. **Platform incompatibility**: The `plugin` package has different behavior across macOS/Linux, and doesn't work at all on Windows
2. **ABI coupling**: Plugins must be compiled with the exact same Go version and dependencies as the main binary
3. **Deployment friction**: Each plugin requires separate build, version management, and distribution
4. **Debugging complexity**: Multi-process debugging, separate logs, harder observability
5. **Security concerns**: Arbitrary code loading at runtime increases attack surface

### Benefits of Internal Plugins

1. **Predictable builds**: Single binary, consistent behavior across platforms
2. **Simple deployment**: One artifact to version and distribute
3. **Unified observability**: Shared logging, tracing, and metrics
4. **Security**: No runtime code loading, all code reviewed at build time
5. **Faster iteration**: Refactor interfaces without rebuilding external plugins

### Tradeoff Accepted

- New plugins require rebuilding and redeploying the binary

This is acceptable because:
- We're in early product stage with frequent architecture changes
- Plugin development happens within the same repo
- Deployment is automated (single binary simplifies CI/CD)

## Consequences

- All plugins live in `internal/plugins/`
- Plugin contract (`Plugin` interface) is defined in `internal/plugins/registry/contracts.go`
- Plugins register at compile time via `RegisterPlugin()` in the registry
- When independent plugin deployment becomes necessary, we can migrate to gRPC sidecars or WASM without changing the event/command contracts

## Evolution Path

When the need arises for independent plugin deployment:
1. Keep event and command contracts stable
2. Move plugin logic to gRPC sidecar services
3. Or use WASM sandbox execution for selected logic
4. The event-driven architecture makes this migration transparent to users
