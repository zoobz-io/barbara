# auth

Resolves the identity and entitlement of an incoming request.

## Purpose

Barbara has no user table, no sessions, no API keys. It delegates identity and
entitlement to [janus](https://github.com/zoobz-io/janus) over the
[aegis](https://github.com/zoobz-io/aegis) mesh, and services authenticate to
each other with mesh CA client certificates (`docs/plans/001-domain.md`,
"Auth"). This package is the seam that resolution flows through — and, until the
mesh integration lands, the stub that stands in for it so surface work is not
blocked.

Every request resolves to a **`Principal`**: the tenant it operates on, the
acting user, and the entitlement (roles/scopes) granted for that tenant.

## Pieces

| Symbol | Role |
|--------|------|
| `Principal` | Barbara's resolved identity; implements `rocco.Identity`, so handlers read it off `req.Identity`. |
| `Authenticator` | The resolution contract: `Authenticate(ctx, *http.Request) (rocco.Identity, error)`. The janus/aegis integration implements this. |
| `StubAuthenticator` | Local-dev / test resolver. Authenticates nothing; resolves every request to a fixed, injected identity. |
| `NewExtractor` | Adapts an `Authenticator` to the function `rocco.Engine.WithAuthenticator` expects. |
| `Wire` | Registers the resolver under `Authenticator` in the sum registry **and** sets it as the engine's extractor. One call per binary. |
| `WithPrincipal` / `PrincipalFromContext` / `TenantFromContext` / `UserFromContext` | Carry the identity down into tenant-scoped stores, which see only a `context.Context`. |

## How a request resolves

1. A handler marks itself `WithAuthentication()`.
2. rocco's auth middleware calls the engine's extractor (installed by `Wire`),
   which delegates to the registered `Authenticator`.
3. On success the returned `Principal` lands on `req.Identity`; on error rocco
   rejects the request with 401.
4. Handlers read `req.Identity` directly. Before calling a **tenant-scoped
   store**, a handler bridges the identity into the context:

   ```go
   ctx := auth.WithPrincipal(req.Context, req.Identity)
   docs, err := documents.List(ctx) // store reads auth.TenantFromContext(ctx)
   ```

   This bridge is needed because rocco keeps its identity under a private
   context key; `auth`'s own key is what a store can read.

## Wiring (per binary)

```go
// after boot.Init, before sum.Freeze
auth.Wire(rt.K, rt.Svc.Engine(), auth.DefaultStub())
```

`cmd/api` already wires the stub. `cmd/admin` wires it identically once that
binary exists.

## Swapping in janus/aegis

The real resolver is a drop-in. It implements `Authenticator` and, per the mesh
contract, must:

- authenticate the caller — mesh client cert for services, a janus session
  token for users;
- determine which tenant the request operates on and verify the caller is
  authorized for it (argus selects the tenant with an `X-Tenant-ID` header; the
  stub honors the same header so call sites don't change);
- return a `Principal` carrying the tenant, the acting user, and the
  tenant-scoped roles/scopes janus reports.

Connection settings for the mesh already live in `config.Mesh`
(`config/mesh.go`). Swapping is a one-line change at the `Wire` call — construct
the mesh resolver instead of `DefaultStub()`. No handler, store, or contract
changes.

The stub is not an authenticator in any security sense and must never be wired
in a deployed binary.

## Internal (admin) vs tenant entitlements

The public API and the admin binary share this stub but stand for different
identities. Tenant users (public API) carry tenant-scoped scopes — `documents:*`
for the authoring lifecycle — and operate within a single tenant. Internal
admin identities are tenant-agnostic and carry a platform entitlement (the
`admin` role) that tenant users never hold; the admin cross-tenant search is
gated on it. When the mesh resolver replaces the stub, it issues each kind of
identity its own entitlements — the swap point in `Wire` is unchanged, and the
handler-level gates (`WithScopes` / `WithRoles`) stay as they are.
