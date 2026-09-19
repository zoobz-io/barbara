import type { components } from "@barbara/api-sdk";

/** A release as the public API serves it. */
export type Release = components["schemas"]["ReleaseResponse"];

/** One live path in a release: key, the document, and the version served. */
export type ReleaseEntry = components["schemas"]["ReleaseEntryResponse"];
