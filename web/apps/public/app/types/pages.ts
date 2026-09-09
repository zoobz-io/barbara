import type { TreeNode } from "@zoobzio/foundation/types/core/tree";

/**
 * A node in the pages tree: a collection (folder) or document (file),
 * carrying its wire id alongside the tree shape. The tree loads whole at
 * page load; an empty folder keeps `children: []` so it stays expandable.
 */
export type PageNode = TreeNode & {
  id: string;
  kind: "folder" | "document";
  children?: PageNode[];
};
