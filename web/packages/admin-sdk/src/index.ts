/**
 * @barbara/admin-sdk — the typed client for the Barbara admin API.
 *
 * `createAdminClient(config)` returns a resource-namespaced client
 * (`client.search.all()`); methods take positional path params then a trailing
 * options object, return the response body directly, and throw from the
 * openapi-press error hierarchy on failure. Cross-cutting behavior (retry,
 * timeouts, caching, …) is applied per endpoint through `.with`.
 */

export { createAdminClient } from "./client";
export type { AdminClient } from "./client";

// Re-export openapi-press's shared surface so consumers configure, instrument,
// and catch without a second dependency.
export {
  withCache,
  withCircuitBreaker,
  withDedupe,
  withFallback,
  withReauth,
  withRetry,
  withTimeout,
} from "openapi-press";
export type {
  CallConfig,
  CallMeta,
  CircuitBreakerPolicy,
  ClientConfig,
  ErrorHookContext,
  Hooks,
  Logger,
  RequestHookContext,
  ResponseHookContext,
  RetryPolicy,
  Wrapper,
} from "openapi-press";
export * from "openapi-press/error";

// The generated schema types, for consumers that want to name request/response
// shapes directly (e.g. `components["schemas"]["SearchResponse"]`).
export type { components, paths } from "./schema";
