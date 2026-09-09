/**
 * ISO timestamp → "Sep 4, 2026, 7:32 PM". Locale and zone are pinned so the
 * server and client render identical text — no hydration mismatch. Viewer
 * locale can take over once formatting moves client-only.
 */
export function formatDate(iso: string): string {
  return new Intl.DateTimeFormat("en-US", {
    dateStyle: "medium",
    timeStyle: "short",
    timeZone: "UTC",
  }).format(new Date(iso));
}

/** A path-like key's file name — its last segment. */
export function keyName(key: string): string {
  return key.split("/").at(-1) ?? key;
}

/** First segment of a UUID — placeholder identity until janus lands. */
export function shortId(id: string): string {
  return id.split("-")[0] ?? id;
}

/** Bytes → "512 B", "4.2 KB", "12 MB". */
export function formatBytes(size: number): string {
  if (size < 1024) return `${size} B`;
  let value = size;
  for (const unit of ["KB", "MB", "GB"]) {
    value /= 1024;
    if (value < 1024) {
      return `${value < 10 ? value.toFixed(1) : Math.round(value)} ${unit}`;
    }
  }
  return `${Math.round(value / 1024)} TB`;
}
