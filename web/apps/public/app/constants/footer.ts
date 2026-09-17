import type { FooterLink } from "~/types/footer";

/**
 * The footer's link line. The GitHub link is real; the rest point at
 * destinations that land with the docs site.
 */
export const FOOTER_LINKS: FooterLink[] = [
  { label: "Docs", to: "/docs" },
  { label: "API", to: "/api" },
  { label: "Status", to: "/status" },
  { label: "GitHub", to: "https://github.com/zoobz-io/barbara", external: true },
];
