import { usePress } from "#imports";

/**
 * The document data layer for editing: open a document with its head
 * content, save a new version against the head the edit was based on.
 * Call in setup; the editor owns fetch timing.
 */
export function useDocumentStore() {
  const api = usePress("api");

  return {
    open: (documentId: string) => api.documents.content(documentId),
    saveVersion: (documentId: string, baseVersion: number, content: string) =>
      api.versions.save(documentId, {
        body: { base_version: baseVersion, content },
      }),
  };
}
