import type { AppContents } from "~/types/apps";
import type { PageNode } from "~/types/pages";
import { keyName } from "~/utils/format";

/**
 * Maps one contents level onto tree nodes: folders first (children filled
 * by the store's walk), then documents labeled by their key's file name
 * and linking to their editor page — navigation is the selection.
 */
export function pageNodes(contents: AppContents, appId: string): PageNode[] {
  return [
    ...contents.subcollections.map(
      (sub): PageNode => ({
        key: sub.id,
        id: sub.id,
        kind: "folder",
        label: sub.name,
        icon: "folder",
        children: [],
      }),
    ),
    ...contents.documents.map(
      (doc): PageNode => ({
        key: doc.id,
        id: doc.id,
        kind: "document",
        label: keyName(doc.key),
        icon: "file",
        link: { to: `/apps/${appId}/content/${doc.id}` },
      }),
    ),
  ];
}
