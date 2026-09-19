import type { BreadcrumbItem } from "@zoobzio/foundation/types/core/breadcrumb";

/**
 * Slash-separated paths, as both the asset tree and the content tree name
 * their folders and files: "" is the root, "guides" a folder beneath it,
 * "guides/install.md" a file in that folder. Every helper is pure; which
 * tree a path belongs to is the caller's business.
 */

/** The path a route's catch-all `path` param names ("" at the root). */
export function folderPath(param: string | string[] | undefined): string {
  if (Array.isArray(param)) return param.join("/");
  return param ?? "";
}

/** The path of a child beneath a folder ("" for the root). */
export function childPath(path: string, name: string): string {
  return path === "" ? name : `${path}/${name}`;
}

/** The folder a path sits in ("" for the root). */
export function parentPath(path: string): string {
  const slash = path.lastIndexOf("/");
  return slash === -1 ? "" : path.slice(0, slash);
}

/**
 * The route for a path beneath a base: each segment encoded on its own so
 * a slash inside a name never reads as a level; "" is the base itself.
 */
export function pathRoute(base: string, path: string): string {
  if (path === "") return base;
  return `${base}/${path.split("/").map(encodeURIComponent).join("/")}`;
}

/**
 * The breadcrumb trail for a path: the root, then one crumb per segment,
 * each linking to its own route. The trailing crumb is the current
 * location; the breadcrumb renders it as the page rather than a link.
 */
export function pathCrumbs(
  path: string,
  rootLabel: string,
  route: (path: string) => string,
): BreadcrumbItem[] {
  const segments = path === "" ? [] : path.split("/");
  return [
    { key: "", label: rootLabel, link: { to: route("") } },
    ...segments.map((label, i) => {
      const key = segments.slice(0, i + 1).join("/");
      return { key, label, link: { to: route(key) } };
    }),
  ];
}
