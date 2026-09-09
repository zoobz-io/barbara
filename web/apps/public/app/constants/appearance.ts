import type { AppearanceGroup } from "~/types/appearance";

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
