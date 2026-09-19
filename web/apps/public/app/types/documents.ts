import type { components } from "@barbara/api-sdk";

/** A document opened for editing: metadata plus its head version content. */
export type DocumentContent =
  components["schemas"]["DocumentContentResponse"];

/** One saved version of a document, as the public API serves it. */
export type Version = components["schemas"]["VersionResponse"];
