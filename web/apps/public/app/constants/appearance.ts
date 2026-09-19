import type { AppearanceGroup, BrandGroup } from "~/types/appearance";

/**
 * Every untheme axis the theme contract offers, grouped for the settings
 * page. Labels are user-facing; contexts are the contract's names.
 */
export const APPEARANCE_GROUPS: AppearanceGroup[] = [
  {
    label: "Colour and contrast",
    settings: [
      {
        modifier: "color",
        label: "Mode",
        description: "Light or dark surfaces",
        options: [
          { label: "Light", context: "light" },
          { label: "Dark", context: "dark" },
        ],
      },
      {
        modifier: "vibrancy",
        label: "Colour strength",
        description: "How saturated accents and badges are",
        options: [
          { label: "Muted", context: "muted" },
          { label: "Normal", context: "balanced" },
          { label: "Vivid", context: "vivid" },
        ],
      },
      {
        modifier: "contrast",
        label: "Contrast",
        description: "Raises text and border contrast",
        options: [
          { label: "Normal", context: "default" },
          { label: "More", context: "medium" },
          { label: "Most", context: "high" },
        ],
      },
    ],
  },
  {
    label: "Layout",
    settings: [
      {
        modifier: "density",
        label: "Spacing",
        description: "How tightly rows and panels pack",
        options: [
          { label: "Compact", context: "compact" },
          { label: "Normal", context: "default" },
          { label: "Roomy", context: "spacious" },
        ],
      },
      {
        modifier: "text",
        label: "Text size",
        description: "Scales every type size together",
        options: [
          { label: "Small", context: "sm" },
          { label: "Normal", context: "md" },
          { label: "Large", context: "lg" },
        ],
      },
      {
        modifier: "radius",
        label: "Corners",
        description: "Sharp is the house style",
        options: [
          { label: "Sharp", context: "sharp" },
          { label: "Soft", context: "default" },
          { label: "Round", context: "round" },
        ],
      },
    ],
  },
  {
    label: "Depth and motion",
    settings: [
      {
        modifier: "depth",
        label: "Shadows",
        description: "Lift on menus, dialogs and buttons",
        options: [
          { label: "None", context: "flat" },
          { label: "Subtle", context: "default" },
          { label: "Strong", context: "deep" },
        ],
      },
      {
        modifier: "motion",
        label: "Animation",
        description: "Transition speed across the app",
        options: [
          { label: "Off", context: "reduced" },
          { label: "Normal", context: "default" },
          { label: "Playful", context: "expressive" },
        ],
      },
    ],
  },
];

/**
 * Every ramp the brand panel exposes, grouped for the settings page. One
 * picked color re-seeds the whole ramp, so every role in the family — fill,
 * text, container, and the contrast and vibrancy channels — follows.
 */
export const BRAND_GROUPS: BrandGroup[] = [
  {
    label: "Brand",
    settings: [
      {
        ramp: "primary",
        swatch: {
          fill: "primary",
          on: "on-primary",
          container: "primary-container",
          onContainer: "on-primary-container",
        },
        label: "Primary",
        description: "Buttons, links, and selected states",
      },
      {
        ramp: "secondary",
        swatch: {
          fill: "secondary",
          on: "on-secondary",
          container: "secondary-container",
          onContainer: "on-secondary-container",
        },
        label: "Secondary",
        description: "Supporting accents and badges",
      },
      {
        ramp: "tertiary",
        swatch: {
          fill: "tertiary",
          on: "on-tertiary",
          container: "tertiary-container",
          onContainer: "on-tertiary-container",
        },
        label: "Tertiary",
        description: "Highlights and decorative accents",
      },
    ],
  },
  {
    label: "Status",
    settings: [
      {
        ramp: "error",
        swatch: {
          fill: "error",
          on: "on-error",
          container: "error-container",
          onContainer: "on-error-container",
        },
        label: "Error",
        description: "Destructive actions and failures",
      },
      {
        ramp: "success",
        swatch: {
          fill: "success",
          on: "on-success",
          container: "success-container",
          onContainer: "on-success-container",
        },
        label: "Success",
        description: "Confirmations and healthy states",
      },
      {
        ramp: "warning",
        swatch: {
          fill: "warning",
          on: "on-warning",
          container: "warning-container",
          onContainer: "on-warning-container",
        },
        label: "Warning",
        description: "Cautions and pending states",
      },
    ],
  },
  {
    label: "Surfaces",
    settings: [
      {
        ramp: "neutral",
        swatch: {
          fill: "surface-container-high",
          on: "on-surface",
          container: "surface",
          onContainer: "on-surface-muted",
        },
        label: "Neutral",
        description: "Backgrounds and body text",
      },
      {
        ramp: "neutral-variant",
        swatch: {
          fill: "outline",
          on: "surface",
          container: "outline-muted",
          onContainer: "on-surface",
        },
        label: "Outline",
        description: "Borders and dividers",
      },
    ],
  },
];
