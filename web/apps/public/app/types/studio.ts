/**
 * A studio tab: a labeled page link, not component state. `match` marks the
 * tab active for any path under the prefix (the Content tab owns its
 * per-page child routes).
 */
export type Tab = {
  label: string;
  to: string;
  match?: string;
};
