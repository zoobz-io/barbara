import type { components } from "@barbara/api-sdk";

/** An asset's metadata as the public API serves it (bytes live elsewhere). */
export type Asset = components["schemas"]["AssetResponse"];

/** An asset as a browser row: the wire shape plus its display name. */
export type AssetRow = Asset & { name: string };
