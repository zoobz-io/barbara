import { usePress } from "#imports";

/**
 * The document data layer for editing: open a document with its head
 * content, list its recent versions, save a new version against the head
 * the edit was based on. Call in setup; the page owns fetch timing.
 */
export function useDocumentStore() {
  const api = usePress("api");

  return {
    open: (documentId: string) => api.documents.content(documentId),
    /** The document's most recent versions, newest first. */
    versions: (documentId: string, limit: number) =>
      api.versions.list(documentId, {
        query: { limit: String(limit), offset: "0" },
      }),
    saveVersion: (documentId: string, baseVersion: number, content: string) =>
      api.versions.save(documentId, {
        body: { base_version: baseVersion, content },
      }),
  };
}
