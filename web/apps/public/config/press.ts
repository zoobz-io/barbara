/**
 * The public-API proxy mount. nuxt.config wires the press client's `prefix`
 * from this constant, and app code builds same-origin asset URLs from it —
 * one source of truth for where the API is mounted in the browser.
 */
export const API_PROXY_PREFIX = "/api";
