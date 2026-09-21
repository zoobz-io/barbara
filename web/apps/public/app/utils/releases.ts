import type { BreadcrumbItem } from "@zoobzio/foundation/types/core/breadcrumb";

import type {
  ChangeKind,
  Release,
  ReleaseChange,
  ReleaseKind,
} from "~/types/releases";
import {
  CHANGE_KINDS,
  CHANGE_LABEL,
  KIND_LABEL,
  RELEASE_ROOT_LABEL,
} from "~/constants/releases";
import { pathRoute } from "~/utils/path";

/** The releases timeline of an app. */
export function releasesRoute(appId: string): string {
  return `/apps/${appId}/releases`;
}

/** One release's page. */
export function releaseRoute(appId: string, releaseId: string): string {
  return `${releasesRoute(appId)}/${releaseId}`;
}

/** The viewer for the page a release served at a key. */
export function releaseEntryRoute(
  appId: string,
  releaseId: string,
  key: string,
): string {
  return pathRoute(releaseRoute(appId, releaseId), key);
}

/** A release's short name: its number, as the timeline headlines it. */
export function releaseTitle(release: Pick<Release, "number">): string {
  return `#${release.number}`;
}

/**
 * The breadcrumb trail into the releases pages: the root, the release, and
 * an optional leaf beneath it (the page a viewer shows). The trailing crumb
 * is the current location; the breadcrumb renders it as the page rather
 * than a link.
 */
export function releaseCrumbs(
  appId: string,
  release?: Pick<Release, "id" | "number">,
  leaf?: string,
): BreadcrumbItem[] {
  const crumbs: BreadcrumbItem[] = [
    { key: "", label: RELEASE_ROOT_LABEL, link: { to: releasesRoute(appId) } },
  ];
  if (release) {
    crumbs.push({
      key: release.id,
      label: releaseTitle(release),
      link: { to: releaseRoute(appId, release.id) },
    });
  }
  if (leaf !== undefined) {
    crumbs.push({ key: `${release?.id ?? ""}:${leaf}`, label: leaf });
  }
  return crumbs;
}

/** True when a string is one of the API's release kinds. */
export function isReleaseKind(kind: string): kind is ReleaseKind {
  return kind in KIND_LABEL;
}

/** The user-facing name of a release kind; an unknown one shows as given. */
export function kindLabel(kind: string): string {
  return isReleaseKind(kind) ? KIND_LABEL[kind] : kind;
}

/** True when a string is one of the API's change kinds. */
export function isChangeKind(change: string): change is ChangeKind {
  return change in CHANGE_LABEL;
}

/** The user-facing name of a change kind; an unknown one shows as given. */
export function changeLabel(change: string): string {
  return isChangeKind(change) ? CHANGE_LABEL[change] : change;
}

/**
 * A release's non-zero change counts against the previous release, in
 * display order — the stats a timeline item shows. A release that changed
 * nothing (a restore of the live release, say) has none.
 */
export function changeCounts(
  release: Pick<Release, ChangeKind>,
): { kind: ChangeKind; count: number }[] {
  return CHANGE_KINDS.map((kind) => ({ kind, count: release[kind] })).filter(
    (c) => c.count > 0,
  );
}

/** A release's change rows by document, for marking the rows of its manifest. */
export function changeByDocument(
  changes: ReleaseChange[],
): Map<string, ReleaseChange> {
  return new Map(changes.map((c) => [c.document_id, c]));
}
