import type { Events as BrowserEvents } from "@zoobzio/foundation/types/data/browser";

/**
 * Foundation's data widgets emit scoped events over Nuxt's hook bus, typed
 * by augmenting RuntimeNuxtHooks. The layer's own app.d.ts does this for its
 * dev app but does not travel with the package, so the consumer re-declares
 * the maps for the widgets it uses.
 */
declare module "#app" {
  interface RuntimeNuxtHooks extends BrowserEvents {}
}

export {};
