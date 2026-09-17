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

/** One unit of the relative scale: its length in ms and its suffix. */
const RELATIVE_UNITS: { ms: number; suffix: string }[] = [
  { ms: 60_000, suffix: "m" },
  { ms: 3_600_000, suffix: "h" },
  { ms: 86_400_000, suffix: "d" },
];

/** Under this age a timestamp reads as "just now"; seconds are not tracked. */
const JUST_NOW_MS = 60_000;

/** Past this age a timestamp shows as its full date, not a relative one. */
const RELATIVE_WINDOW_MS = 7 * 86_400_000;

/**
 * ISO timestamp → how long ago, for recent times: "just now" (under a
 * minute), "25m ago", "3h ago", "2d ago". Past a week it falls back to the full date,
 * since "12d ago" reads worse than the day itself. Each unit takes over at
 * its own length (60m → "1h ago") and truncates rather than rounds. A time
 * in the future (clock skew) is "just now"; an unparseable one comes back
 * as given.
 *
 * `now` is the reference in ms; pass `useNow()` so the server and client
 * render the same text at hydration and the client ticks afterwards.
 */
export function formatRelative(iso: string, now: number = Date.now()): string {
  const age = now - new Date(iso).getTime();
  if (Number.isNaN(age)) return iso;
  if (age < JUST_NOW_MS) return "just now";
  if (age >= RELATIVE_WINDOW_MS) return formatDate(iso);
  let unit = RELATIVE_UNITS[0]!;
  for (const next of RELATIVE_UNITS) {
    if (age < next.ms) break;
    unit = next;
  }
  return `${Math.floor(age / unit.ms)}${unit.suffix} ago`;
}

/** A path-like key's file name — its last segment. */
export function keyName(key: string): string {
  return key.split("/").at(-1) ?? key;
}

/** First segment of a UUID — placeholder identity until janus lands. */
export function shortId(id: string): string {
  return id.split("-")[0] ?? id;
}

/**
 * Bytes → "512 B", "4.2 KB", "12 MB". A value that would round up to its
 * unit's ceiling moves up a unit instead: 1023.6 MB is "1.0 GB", never
 * "1024 MB".
 */
export function formatBytes(size: number): string {
  if (size < 1024) return `${size} B`;
  let value = size;
  for (const unit of ["KB", "MB", "GB"]) {
    value /= 1024;
    if (value < 1023.5) {
      return `${value < 10 ? value.toFixed(1) : Math.round(value)} ${unit}`;
    }
  }
  return `${Math.round(value / 1024)} TB`;
}

/** A count with its noun: "1 file", "3 files". */
export function counted(n: number, noun: string, plural = `${noun}s`): string {
  return `${n} ${n === 1 ? noun : plural}`;
}
