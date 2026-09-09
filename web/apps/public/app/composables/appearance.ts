import { useUntheme } from "#imports";

import type { AppearanceModifier } from "~/types/appearance";

/**
 * The untheme selection as appearance state: read the active context per
 * modifier, swap to another. Persistence rides untheme's cookie.
 */
export function useAppearance() {
  const ut = useUntheme();

  return {
    active: (modifier: AppearanceModifier): string =>
      ut.config.input[modifier],
    // One contained cast: callers iterate the settings union, which TS
    // cannot correlate back to swap's paired generics.
    set: (modifier: AppearanceModifier, context: string) => {
      ut.swap(modifier, context as never);
    },
  };
}
