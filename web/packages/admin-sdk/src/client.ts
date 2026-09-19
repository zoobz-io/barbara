/**
 * The admin client factory. Assembles the namespaces into one tree and captures
 * it into a Press via `client` from the spec binding. The returned client is
 * fully typed against the admin spec; `createAdminClient` is the only entry
 * point. The surface is deliberately small — cross-tenant search is the one
 * capability seeded here; the platform grows from this.
 */

import type { paths } from "./schema";

import { definePress } from "openapi-press";

const { op, client } = definePress<paths>();

/**
 * Creates an admin client. With no config it targets the same origin (relative
 * `baseUrl`), which is what the browser wants when a proxy fronts the admin API.
 */
export const createAdminClient = client({
  search: {
    all: op("get", "/search"),
  },
});

/** A live, fully-typed admin client. */
export type AdminClient = ReturnType<typeof createAdminClient>;
