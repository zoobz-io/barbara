import { usePress } from "#imports";

/**
 * The document data layer for editing: open a document with its head
 * content, read one version or the document's row, list its recent
 * versions, save a new version against the head the edit was based on.
 * Call in setup; the page owns fetch timing.
 */
export function useDocumentStore() {
  const api = usePress("api");

  return {
    open: (documentId: string) => api.documents.content(documentId),
    /** One saved version, with its content. */
    version: (versionId: string) => api.versions.get(versionId),
    /** A document's row: its current key and status. */
    get: (documentId: string) => api.documents.get(documentId),
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
