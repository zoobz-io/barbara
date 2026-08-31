import { defineNuxtConfig } from "nuxt/config";

export default defineNuxtConfig({
  compatibilityDate: "2026-08-31",
  // Fixed port so both apps run side by side (admin sits on 3001).
  devServer: { port: 3000 },
  runtimeConfig: {
    barbara: {
      // The public API this site fronts.
      apiHost: "http://127.0.0.1:8080",
    },
  },
});
