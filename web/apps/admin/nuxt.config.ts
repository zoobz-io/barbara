import { defineNuxtConfig } from "nuxt/config";

export default defineNuxtConfig({
  compatibilityDate: "2026-08-31",
  // Fixed port so both apps run side by side (public sits on 3000).
  devServer: { port: 3001 },
  runtimeConfig: {
    barbara: {
      // The admin API this console fronts.
      adminHost: "http://127.0.0.1:8081",
    },
  },
});
