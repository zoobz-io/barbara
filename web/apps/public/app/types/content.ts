import type { TreeNode } from "@zoobzio/foundation/types/core/tree";
import type { components } from "@barbara/api-sdk";

/** A collection (folder in the content tree) as the public API serves it. */
export type Collection = components["schemas"]["CollectionResponse"];

/**
 * A document (page in the content tree) as the public API serves it: its
 * key is the full path, its status derives from the app's current release.
 */
export type Document = components["schemas"]["DocumentResponse"];

/** A document's lifecycle status, as the API names it. */
export type DocumentStatus =
  "draft" | "published" | "published-with-newer-draft";

/**
 * One level of an app's content tree: the collection it is (null at the
 * app root), its direct subcollections, and its direct documents. The API
 * addresses levels by collection id; the store resolves each from its
 * path, so pages can be path-addressed like assets.
 */
export type ContentLevel = {
  id: string | null;
  subcollections: Collection[];
  documents: Document[];
};

/**
 * A node in the pages sidebar tree: a collection (folder) or document
 * (file), carrying its path and wire id alongside the tree shape. The tree
 * loads whole; an empty folder keeps `children: []` so it stays expandable.
 */
export type ContentNode = TreeNode & {
  id: string;
  path: string;
  kind: "folder" | "document";
  children?: ContentNode[];
};

/** How a folder's rows order: by name or by when they last changed. */
export type ContentSort = "name" | "updated";
