import { onMounted, useState } from "#imports";

/** How often the shared clock advances on the client: the finest unit the
 * relative format shows is the minute, so a tick every 30 seconds keeps it
 * at most half a minute stale. */
const TICK_MS = 30_000;

/** One interval per client, however many components read the clock. */
let ticking = false;

/**
 * The current time in ms, as shared state: taken once on the server so
 * every relative time hydrates with the text the server rendered, then
 * advanced on the client every half minute so "just now" becomes
 * "1m ago" without a reload. Read it wherever a relative time renders. Call in
 * setup.
 */
export function useNow() {
  const now = useState<number>("clock:now", () => Date.now());
  onMounted(() => {
    if (ticking) return;
    ticking = true;
    now.value = Date.now();
    setInterval(() => {
      now.value = Date.now();
    }, TICK_MS);
  });
  return now;
}
