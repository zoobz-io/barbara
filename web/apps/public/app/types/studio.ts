import type { IconAlias } from "@zoobzio/foundation/types/icon";

/**
 * A studio tab: an icon-led page link, not component state. `match` marks
 * the tab active for any path under the prefix (the Content tab owns its
 * per-page child routes; Assets owns every folder and asset beneath it).
 */
export type Tab = {
  icon: IconAlias;
  label: string;
  to: string;
  match?: string;
};
