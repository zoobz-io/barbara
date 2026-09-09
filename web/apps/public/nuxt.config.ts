import { defineNuxtConfig } from "nuxt/config";

import iconSheets from "./config/icon-sheets";
import { API_PROXY_PREFIX } from "./config/press";

export default defineNuxtConfig({
  compatibilityDate: "2026-08-31",
  extends: ["@zoobzio/foundation"],
  css: ["~/assets/css/main.css"],
  iconSheets,
  devServer: { port: 3000 },
  modules: ["@openapi-press/nuxt"],
  press: {
    clients: {
      api: {
        client: "@barbara/api-sdk",
        host: "http://127.0.0.1:8080",
        prefix: API_PROXY_PREFIX,
      },
    },
  },
});
