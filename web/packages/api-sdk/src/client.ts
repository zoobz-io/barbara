/**
 * The public-API client factory. Assembles the domain namespaces into one tree
 * and captures it into a Press via `client` from the spec binding. The returned
 * client is fully typed against the public API spec; `createApiClient` is the
 * only entry point.
 */

import type { paths } from "./schema";

import { definePress } from "openapi-press";

const { op, client } = definePress<paths>();

/**
 * Creates a public-API client. With no config it targets the same origin
 * (relative `baseUrl`), which is what the browser wants when a proxy fronts
 * the API.
 */
export const createApiClient = client({
  // The site-facing published read surface, served from the search index.
  published: {
    lookup: op("get", "/published/apps/{app_id}/lookup"),
    enumerate: op("get", "/published/apps/{app_id}/documents"),
    folder: op("get", "/published/apps/{app_id}/folder"),
    search: op("get", "/published/apps/{app_id}/search"),
    asset: op("get", "/published/apps/{app_id}/assets/object"),
  },

  apps: {
    list: op("get", "/apps"),
    create: op("post", "/apps"),
    get: op("get", "/apps/{id}"),
    rename: op("patch", "/apps/{id}"),
    delete: op("delete", "/apps/{id}"),
    contents: op("get", "/apps/{app_id}/contents"),
  },

  collections: {
    create: op("post", "/apps/{app_id}/collections"),
    get: op("get", "/apps/{app_id}/collections/{id}"),
    contents: op("get", "/apps/{app_id}/collections/{id}/contents"),
    rename: op("patch", "/apps/{app_id}/collections/{id}"),
    move: op("post", "/apps/{app_id}/collections/{id}/move"),
    delete: op("delete", "/apps/{app_id}/collections/{id}"),
  },

  documents: {
    list: op("get", "/documents"),
    create: op("post", "/apps/{app_id}/documents"),
    get: op("get", "/documents/{id}"),
    content: op("get", "/documents/{id}/content"),
    move: op("post", "/apps/{app_id}/documents/{id}/move"),
    delete: op("delete", "/documents/{id}"),

    tags: {
      add: op("post", "/documents/{id}/tags"),
      remove: op("delete", "/documents/{id}/tags"),
    },

    // Per-document publishing lifecycle.
    publish: op("post", "/documents/{id}/publish"),
    unpublish: op("post", "/documents/{id}/unpublish"),
    rollback: op("post", "/documents/{id}/rollback"),
  },

  versions: {
    save: op("post", "/documents/{document_id}/versions"),
    list: op("get", "/documents/{document_id}/versions"),
    get: op("get", "/versions/{id}"),
  },

  releases: {
    cut: op("post", "/apps/{app_id}/releases"),
    list: op("get", "/apps/{app_id}/releases"),
    get: op("get", "/apps/{app_id}/releases/{id}"),
    rollback: op("post", "/apps/{app_id}/releases/{id}/rollback"),
  },

  assets: {
    list: op("get", "/apps/{app_id}/assets"),
    upload: op("put", "/apps/{app_id}/assets/object"),
    get: op("get", "/apps/{app_id}/assets/object"),
    delete: op("delete", "/apps/{app_id}/assets/object"),
  },
});

/** A live, fully-typed public-API client. */
export type ApiClient = ReturnType<typeof createApiClient>;
