import type { components } from "@barbara/api-sdk";

/**
 * A release as the public API serves it: its identity, what kind of cut it
 * was, and its counts against the release before it.
 */
export type Release = components["schemas"]["ReleaseResponse"];

/** One live path in a release: key, the document, and the version served. */
export type ReleaseEntry = components["schemas"]["ReleaseEntryResponse"];

/** One document that differs between a release and the one before it. */
export type ReleaseChange = components["schemas"]["ReleaseChangeResponse"];

/** What cut a release, as the API names it. */
export type ReleaseKind = "cut" | "publish" | "unpublish" | "rollback";

/** How a document differs from the previous release, as the API names it. */
export type ChangeKind = "added" | "changed" | "removed" | "moved";
