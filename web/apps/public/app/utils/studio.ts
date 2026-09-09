import type { MenuGroup } from "@zoobzio/foundation/types/core/menu";

import type { App } from "~/types/apps";
import type { Tab } from "~/types/studio";
import { ALL_APPS_LABEL } from "~/constants/studio";

/** The studio tab links for an app. */
export function appTabs(id: string): Tab[] {
  return [
    { label: "Content", to: `/apps/${id}`, match: `/apps/${id}/content/` },
    { label: "Assets", to: `/apps/${id}/assets` },
    { label: "History", to: `/apps/${id}/history` },
    { label: "Settings", to: `/apps/${id}/settings` },
  ];
}

/** Whether a tab is active for the current path. */
export function tabActive(tab: Tab, path: string): boolean {
  return path === tab.to || (tab.match != null && path.startsWith(tab.match));
}

/**
 * The app-picker menu groups: the other apps to switch to, then navigation
 * back to the landing page.
 */
export function appMenuGroups(apps: App[], currentId: string): MenuGroup[] {
  const others = apps.filter((a) => a.id !== currentId);
  return [
    ...(others.length
      ? [
          {
            key: "apps",
            label: "Switch app",
            items: others.map((a) => ({ label: a.name })),
          },
        ]
      : []),
    { key: "nav", items: [{ label: ALL_APPS_LABEL }] },
  ];
}
