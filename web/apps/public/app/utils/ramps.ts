import type { Color } from "untheme";
import type { AppUnthemeThemeLayer } from "#imports";

import type { BrandRamp, BrandSeeds } from "~/types/appearance";

/**
 * Aurora's tonal ramp generator, ported to run in the browser. The preset
 * ships its ramps as build-time literals produced by this same math
 * (presets/aurora/scripts/generate.mjs): a seed contributes hue and chroma
 * in OKLCH, every ramp shares one lightness ladder across the eleven stops,
 * and a chroma curve peaks at the middle and tapers toward both ends. The
 * accent ramps also carry muted and vivid columns for the vibrancy axis.
 * Because every color role references a ramp stop, regenerating a ramp
 * re-tints the whole role family — fill, text, container, contrast and
 * vibrancy channels — in every mode.
 */

/** The ramps that carry only the balanced column. */
const NEUTRAL_RAMPS: ReadonlySet<BrandRamp> = new Set(["neutral", "neutral-variant"]);

const COLUMNS = [
  { suffix: "", chroma: 1 },
  { suffix: "-muted", chroma: 0.45 },
  { suffix: "-vivid", chroma: 1.4 },
] as const;

const STOPS = {
  50: { lightness: 0.975, chroma: 0.22 },
  100: { lightness: 0.945, chroma: 0.38 },
  200: { lightness: 0.885, chroma: 0.6 },
  300: { lightness: 0.805, chroma: 0.82 },
  400: { lightness: 0.715, chroma: 0.95 },
  500: { lightness: 0.62, chroma: 1 },
  600: { lightness: 0.53, chroma: 1 },
  700: { lightness: 0.45, chroma: 0.94 },
  800: { lightness: 0.375, chroma: 0.84 },
  900: { lightness: 0.3, chroma: 0.7 },
  950: { lightness: 0.245, chroma: 0.55 },
} as const;

type Oklch = { lightness: number; chroma: number; hue: number };

/* ── sRGB ↔ OKLCH ────────────────────────────────────────────────────── */

const linear = (channel: number): number => {
  if (channel <= 0.04045) {
    return channel / 12.92;
  }
  return ((channel + 0.055) / 1.055) ** 2.4;
};

const gamma = (channel: number): number => {
  if (channel <= 0.0031308) {
    return 12.92 * channel;
  }
  return 1.055 * channel ** (1 / 2.4) - 0.055;
};

/** A `#rrggbb` color's OKLCH coordinates. */
const oklch = (hex: string): Oklch => {
  const [r, g, b] = [1, 3, 5].map((at) => {
    return linear(Number.parseInt(hex.slice(at, at + 2), 16) / 255);
  }) as [number, number, number];
  const l = Math.cbrt(0.4122214708 * r + 0.5363325363 * g + 0.0514459929 * b);
  const m = Math.cbrt(0.2119034982 * r + 0.6806995451 * g + 0.1073969566 * b);
  const s = Math.cbrt(0.0883024619 * r + 0.2817188376 * g + 0.6299787005 * b);
  const a = 1.9779984951 * l - 2.428592205 * m + 0.4505937099 * s;
  const b2 = 0.0259040371 * l + 0.7827717662 * m - 0.808675766 * s;
  return {
    lightness: 0.2104542553 * l + 0.793617785 * m - 0.0040720468 * s,
    chroma: Math.hypot(a, b2),
    hue: Math.atan2(b2, a),
  };
};

/**
 * OKLCH coordinates as sRGB channels in [0, 1], or null when the color
 * falls outside the sRGB gamut.
 */
const srgb = ({ lightness, chroma, hue }: Oklch): number[] | null => {
  const a = chroma * Math.cos(hue);
  const b = chroma * Math.sin(hue);
  const l = (lightness + 0.3963377774 * a + 0.2158037573 * b) ** 3;
  const m = (lightness - 0.1055613458 * a - 0.0638541728 * b) ** 3;
  const s = (lightness - 0.0894841775 * a - 1.291485548 * b) ** 3;
  const channels = [
    +4.0767416621 * l - 3.3077115913 * m + 0.2309699292 * s,
    -1.2684380046 * l + 2.6097574011 * m - 0.3413193965 * s,
    -0.0041960863 * l - 0.7034186147 * m + 1.707614701 * s,
  ].map(gamma);
  if (channels.some((channel) => channel < -0.0001 || channel > 1.0001)) {
    return null;
  }
  return channels.map((channel) => Math.min(1, Math.max(0, channel)));
};

/**
 * The nearest in-gamut sRGB channels: chroma reduces — lightness and hue
 * hold — until sRGB can express the color.
 */
const fit = ({ lightness, chroma, hue }: Oklch): number[] => {
  const direct = srgb({ lightness, chroma, hue });
  if (direct) {
    return direct;
  }
  let low = 0;
  let high = chroma;
  let best = srgb({ lightness, chroma: 0, hue }) ?? [0, 0, 0];
  for (let step = 0; step < 24; step++) {
    const middle = (low + high) / 2;
    const attempt = srgb({ lightness, chroma: middle, hue });
    if (attempt) {
      best = attempt;
      low = middle;
    } else {
      high = middle;
    }
  }
  return best;
};

/* ── emission ────────────────────────────────────────────────────────── */

/** A structured srgb color from unit channels, with its hex fallback. */
const color = (channels: number[]): Color => {
  const bytes = channels.map((channel) => Math.round(channel * 255));
  const hex = bytes.map((byte) => byte.toString(16).padStart(2, "0")).join("");
  return {
    colorSpace: "srgb",
    components: bytes.map((byte) => Math.round((byte / 255) * 1e5) / 1e5),
    hex: `#${hex}`,
  };
};

/** Whether a string is a `#rrggbb` color, the form the native picker emits. */
export const isHex = (value: string): value is `#${string}` =>
  /^#[0-9a-f]{6}$/i.test(value);

/**
 * One ramp's token bindings from a seed: the seed's hue and chroma carried
 * across the lightness ladder in each of the ramp's columns.
 */
export const ramp = (
  name: BrandRamp,
  seed: string,
): NonNullable<AppUnthemeThemeLayer["tokens"]> => {
  const { chroma, hue } = oklch(seed.toLowerCase());
  const columns = NEUTRAL_RAMPS.has(name) ? COLUMNS.slice(0, 1) : COLUMNS;
  const tokens: Record<string, Color> = {};
  for (const column of columns) {
    for (const [stop, curve] of Object.entries(STOPS)) {
      tokens[`${name}${column.suffix}-${stop}`] = color(
        fit({
          lightness: curve.lightness,
          chroma: chroma * curve.chroma * column.chroma,
          hue,
        }),
      );
    }
  }
  // The keys are the contract's own ramp stop names; one contained cast
  // where the template-string construction meets the token union.
  return tokens as NonNullable<AppUnthemeThemeLayer["tokens"]>;
};

/**
 * A brand as a theme layer: the ramps re-seeded from the picked colors,
 * nothing else. Ramps without a seed keep the base theme's values, exactly
 * as the preset's own theme variants do.
 */
export const brandLayer = (seeds: BrandSeeds): AppUnthemeThemeLayer => {
  const tokens: NonNullable<AppUnthemeThemeLayer["tokens"]> = {};
  for (const [name, seed] of Object.entries(seeds)) {
    if (seed && isHex(seed)) {
      Object.assign(tokens, ramp(name as BrandRamp, seed));
    }
  }
  return { id: "brand", name: "Brand", tokens };
};
