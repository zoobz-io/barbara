import type { Option } from "@zoobzio/foundation/types/core/common";

import type { ContentSort, DocumentStatus } from "~/types/content";

/** The root crumb of the content tree. */
export const CONTENT_ROOT_LABEL = "Content";

/** The landing page's one-line description of the feature. */
export const CONTENT_LEDE =
  "The pages of your site, in folders. Open a folder to browse it and a page to edit it; every save is a new version, and nothing goes live until a release is cut.";

/** The user-facing name of each status. */
export const STATUS_LABEL: Record<DocumentStatus, string> = {
  draft: "Draft",
  published: "Published",
  "published-with-newer-draft": "Published, newer draft",
};

/** The sort choices of the folder toolbar, as the select lists them. */
export const CONTENT_SORT_OPTIONS: (Option & { value: ContentSort })[] = [
  { value: "name", label: "Name" },
  { value: "updated", label: "Updated" },
];

/** The sort a folder opens with. */
export const DEFAULT_CONTENT_SORT: ContentSort = "name";

/** Versions the editor's side panel lists, newest first. */
export const VERSION_PANEL_CAP = 10;
