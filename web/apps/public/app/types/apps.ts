import type { components } from "@barbara/api-sdk";

/** An app as the public API serves it. */
export type App = components["schemas"]["AppResponse"];

/** A tree level: direct subcollections and documents together. */
export type AppContents = components["schemas"]["CollectionContentsResponse"];
