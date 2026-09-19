import type { Mod } from "#build/types/untheme";

/** A theme modifier axis surfaced in the appearance settings. */
export type AppearanceModifier = keyof Mod & string;

/**
 * A segmented appearance setting: one modifier axis, its user-facing copy,
 * and the ordered options mapping labels to the contract's context names.
 * The mapped-union shape keeps each setting's options typed against its own
 * modifier's contexts.
 */
export type AppearanceSetting = {
  [M in AppearanceModifier]: {
    modifier: M;
    label: string;
    description: string;
    options: { label: string; context: keyof Mod[M] & string }[];
  };
}[AppearanceModifier];

/** A titled group of appearance settings. */
export type AppearanceGroup = {
  label: string;
  settings: AppearanceSetting[];
};

/** A tonal ramp the brand panel re-seeds from a picked color. */
export type BrandRamp =
  | "primary"
  | "secondary"
  | "tertiary"
  | "error"
  | "success"
  | "warning"
  | "neutral"
  | "neutral-variant";

/** The picked seed per ramp; a missing ramp keeps the base theme's values. */
export type BrandSeeds = Partial<Record<BrandRamp, string>>;

/** A brand picker row: one ramp, its user-facing copy. */
export type BrandSetting = {
  ramp: BrandRamp;
  label: string;
  description: string;
  /** The role tokens the preview swatches paint: fill on text, container on text. */
  swatch: { fill: string; on: string; container: string; onContainer: string };
};

/** A titled group of brand pickers. */
export type BrandGroup = {
  label: string;
  settings: BrandSetting[];
};
