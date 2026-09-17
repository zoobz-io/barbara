import type { MenuGroup } from "@zoobzio/foundation/types/core/menu";

import type { App } from "~/types/apps";
import type { Tab } from "~/types/studio";

/** The studio tab links for an app. */
export function appTabs(id: string): Tab[] {
  return [
    {
      icon: "file-text",
      label: "Content",
      to: `/apps/${id}`,
      match: `/apps/${id}/content/`,
    },
    {
      icon: "image",
      label: "Assets",
      to: `/apps/${id}/assets`,
      match: `/apps/${id}/assets/`,
    },
    { icon: "history", label: "History", to: `/apps/${id}/history` },
    { icon: "settings", label: "Settings", to: `/apps/${id}/settings` },
  ];
}

/** Whether a tab is active for the current path. */
export function tabActive(tab: Tab, path: string): boolean {
  return path === tab.to || (tab.match != null && path.startsWith(tab.match));
}

/**
 * The app-picker menu groups: the other apps to switch to. Navigation back
 * to the landing page is the top bar's wordmark, not a menu item.
 */
export function appMenuGroups(apps: App[], currentId: string): MenuGroup[] {
  const others = apps.filter((a) => a.id !== currentId);
  return [
    {
      key: "apps",
      label: "Switch app",
      items: others.length
        ? others.map((a) => ({ label: a.name }))
        : [{ label: "No other apps", disabled: true }],
    },
  ];
}
