import { defineConfig } from "vitest/config";

/**
 * Whole-repo entry point: each package carries its own vitest.config.ts, and
 * this config federates them as projects so a root `vitest` run (IDE, CI,
 * aggregated coverage) still covers everything. Day-to-day runs go through
 * `pnpm test`, which fans out to the packages in parallel.
 */
export default defineConfig({
  test: {
    projects: ["packages/*/vitest.config.ts"],
    coverage: {
      provider: "v8",
      reporter: ["text", "html", "lcov"],
      reportsDirectory: ".coverage",
      include: ["packages/*/src/**/*.ts"],
    },
  },
});
