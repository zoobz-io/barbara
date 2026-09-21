import type { ChangeKind, ReleaseKind } from "~/types/releases";

/** Releases fetched per page of the timeline; older pages append on demand. */
export const RELEASE_PAGE_SIZE = 10;

/** The root crumb of the releases pages. */
export const RELEASE_ROOT_LABEL = "Releases";

/** The landing page's one-line description of the feature. */
export const RELEASES_LEDE =
  "Every release is a snapshot of the pages that were live, numbered in order and never changed. Open one to see what it served, and restore it as a new release.";

/** The user-facing name of each release kind. */
export const KIND_LABEL: Record<ReleaseKind, string> = {
  cut: "Release",
  publish: "Page published",
  unpublish: "Page unpublished",
  rollback: "Restored",
};

/** The user-facing name of each change kind, in display order. */
export const CHANGE_LABEL: Record<ChangeKind, string> = {
  added: "Added",
  changed: "Changed",
  removed: "Removed",
  moved: "Moved",
};

/** The change kinds in the order the counts read. */
export const CHANGE_KINDS: ChangeKind[] = ["added", "changed", "removed", "moved"];
