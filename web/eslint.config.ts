import eslint from "@eslint/js";
import tseslint from "typescript-eslint";
import prettierPlugin from "eslint-plugin-prettier/recommended";

export default tseslint.config(
  eslint.configs.recommended,
  tseslint.configs.recommended,
  prettierPlugin,
  {
    ignores: [
      "**/dist/",
      "**/.dist/",
      "**/.nuxt/",
      "**/.output/",
      "**/.data/",
      "**/*.gen.ts",
      // Generated openapi-typescript client types.
      "packages/*/src/schema.ts",
      // The Nuxt apps carry their own nuxi/vue-tsc tooling.
      "apps/**",
    ],
  },
  {
    rules: {
      "@typescript-eslint/no-unused-vars": [
        "error",
        { argsIgnorePattern: "^_" },
      ],
      "@typescript-eslint/consistent-type-imports": "error",
    },
  },
);
