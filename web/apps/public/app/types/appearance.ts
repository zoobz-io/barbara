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
