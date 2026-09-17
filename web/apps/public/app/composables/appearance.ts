import { useState, useUntheme } from "#imports";

import type {
  AppearanceModifier,
  BrandRamp,
  BrandSeeds,
} from "~/types/appearance";
import { brandLayer } from "~/utils/ramps";

/**
 * The untheme selection as appearance state: read the active context per
 * modifier, swap to another. Persistence rides untheme's cookie.
 *
 * The brand pickers ride the same service: a picked seed regenerates its
 * ramp and applies the result as a theme layer, so every role that
 * references the ramp re-tints in place. The seeds live in app state for
 * the session; persisting them lands with site metadata on the API.
 */
export function useAppearance() {
  const ut = useUntheme();
  const seeds = useState<BrandSeeds>("appearance:brand", () => ({}));

  /**
   * The seed the picker shows for a ramp: the picked color, else the ramp's
   * 600 stop — the one stop that carries the seed's hue and chroma
   * unscaled, so it reproduces the ramp when fed back through the picker.
   */
  const seed = (ramp: BrandRamp): string => {
    const picked = seeds.value[ramp];
    if (picked) {
      return picked;
    }
    const stop = ut.resolve(`${ramp}-600`);
    if (typeof stop === "object" && stop !== null && "hex" in stop) {
      return String(stop.hex);
    }
    return "#000000";
  };

  return {
    active: (modifier: AppearanceModifier): string =>
      ut.config.input[modifier],
    // One contained cast: callers iterate the settings union, which TS
    // cannot correlate back to swap's paired generics.
    set: (modifier: AppearanceModifier, context: string) => {
      ut.swap(modifier, context as never);
    },
    seed,
    branded: (ramp: BrandRamp): boolean => seeds.value[ramp] !== undefined,
    brand: (ramp: BrandRamp, hex: string) => {
      seeds.value = { ...seeds.value, [ramp]: hex };
      ut.apply(brandLayer(seeds.value));
    },
    unbrand: (ramp: BrandRamp) => {
      const { [ramp]: _, ...rest } = seeds.value;
      seeds.value = rest;
      ut.apply(brandLayer(seeds.value));
    },
    resetBrand: () => {
      seeds.value = {};
      ut.apply(brandLayer({}));
    },
  };
}
