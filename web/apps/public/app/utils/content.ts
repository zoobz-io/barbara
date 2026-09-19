import type { BreadcrumbItem } from "@zoobzio/foundation/types/core/breadcrumb";

import type {
  Collection,
  ContentLevel,
  ContentNode,
  ContentSort,
  Document,
  DocumentStatus,
} from "~/types/content";
import { STATUS_LABEL } from "~/constants/content";
import { keyName } from "~/utils/format";
import { childPath, pathCrumbs, pathRoute } from "~/utils/path";

/**
 * The route for a path under an app's content — a folder or a document
 * key, the page tells them apart; "" is the root.
 */
export function contentRoute(appId: string, path: string): string {
  return pathRoute(`/apps/${appId}/content`, path);
}

/** The breadcrumb trail for a path under an app's content. */
export function contentCrumbs(
  appId: string,
  path: string,
  rootLabel: string,
): BreadcrumbItem[] {
  return pathCrumbs(path, rootLabel, (p) => contentRoute(appId, p));
}

/** True when a string is one of the API's document statuses. */
export function isDocumentStatus(status: string): status is DocumentStatus {
  return status in STATUS_LABEL;
}

/** The user-facing name of a status; an unknown one shows as given. */
export function statusLabel(status: string): string {
  return isDocumentStatus(status) ? STATUS_LABEL[status] : status;
}

/**
 * Maps one level onto sidebar tree nodes: folders first (children filled
 * by the store's walk), then documents labeled by their key's file name
 * and linking to their page — navigation is the selection.
 */
export function contentNodes(
  level: ContentLevel,
  appId: string,
  path: string,
): ContentNode[] {
  return [
    ...level.subcollections.map((sub): ContentNode => ({
      key: sub.id,
      id: sub.id,
      path: childPath(path, sub.name),
      kind: "folder",
      label: sub.name,
      icon: "folder",
      children: [],
    })),
    ...level.documents.map((doc): ContentNode => ({
      key: doc.id,
      id: doc.id,
      path: doc.key,
      kind: "document",
      label: keyName(doc.key),
      icon: "file",
      link: { to: contentRoute(appId, doc.key) },
    })),
  ];
}

/** True when a row's name contains the query, case-insensitively. */
function matchesQuery(name: string, query: string): boolean {
  const q = query.trim().toLowerCase();
  return q === "" || name.toLowerCase().includes(q);
}

/**
 * Orders a level's rows. Name sorts A to Z; updated sorts newest first,
 * since the recent change is what the reader is after when they pick it.
 * Ties fall back to the name, so the order is stable.
 */
function compareRows(
  sort: ContentSort,
  a: { name: string; at: string },
  b: { name: string; at: string },
): number {
  const byName = a.name.localeCompare(b.name, "en", { sensitivity: "base" });
  if (sort === "updated") return b.at.localeCompare(a.at) || byName;
  return byName;
}

/** The subcollections matching the query, in sort order. */
export function sortCollections(
  subs: Collection[],
  sort: ContentSort,
  query = "",
): Collection[] {
  return subs
    .filter((c) => matchesQuery(c.name, query))
    .sort((a, b) =>
      compareRows(
        sort,
        { name: a.name, at: a.updated_at },
        { name: b.name, at: b.updated_at },
      ),
    );
}

/** The documents matching the query by file name, in sort order. */
export function sortDocuments(
  docs: Document[],
  sort: ContentSort,
  query = "",
): Document[] {
  return docs
    .filter((d) => matchesQuery(keyName(d.key), query))
    .sort((a, b) =>
      compareRows(
        sort,
        { name: keyName(a.key), at: a.updated_at },
        { name: keyName(b.key), at: b.updated_at },
      ),
    );
}

/**
 * The keys of the folders on the way to a path: every folder node whose
 * path is a strict prefix of it, top down. Expanding these makes the node
 * at the path visible in the sidebar tree.
 */
export function ancestorKeys(nodes: ContentNode[], path: string): string[] {
  const keys: string[] = [];
  for (const node of nodes) {
    if (node.kind !== "folder" || !path.startsWith(`${node.path}/`)) continue;
    keys.push(node.key, ...ancestorKeys(node.children ?? [], path));
  }
  return keys;
}

/** The node at a path, searched depth-first; undefined when none is. */
export function findNode(
  nodes: ContentNode[],
  path: string,
): ContentNode | undefined {
  for (const node of nodes) {
    if (node.path === path) return node;
    const hit = findNode(node.children ?? [], path);
    if (hit) return hit;
  }
  return undefined;
}
