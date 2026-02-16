import prettier from "eslint-config-prettier";
import { includeIgnoreFile } from "@eslint/compat";
import js from "@eslint/js";
import svelte from "eslint-plugin-svelte";
import globals from "globals";
import { fileURLToPath } from "node:url";
import { defineConfig } from "eslint/config";
import svelteConfig from "./svelte.config.js";
import ts from "typescript-eslint";

const gitignorePath = fileURLToPath(new URL("./.gitignore", import.meta.url));

export default defineConfig(
    includeIgnoreFile(gitignorePath),
    js.configs.recommended,
    ...ts.configs.recommended,
    ...svelte.configs.recommended,
    prettier,
    ...svelte.configs.prettier,
    {
        languageOptions: { globals: { ...globals.browser, ...globals.node } },
        rules: {
            "no-console": "error",
            "no-restricted-imports": [
                "error",
                {
                    paths: [
                        {
                            name: "$lib/api/api-helpers",
                            message:
                                "Please use specialized API objects or Providers. Raw helpers are only allowed in the API/Provider layer.",
                        },
                    ],
                    patterns: [
                        {
                            group: ["$lib/api/auth/*"],
                            message: "Please use AuthProvider instead unless absolutely necessary.",
                        },
                    ],
                },
            ],

            // typescript-eslint strongly recommend that you do not use the no-undef lint rule on TypeScript projects.
            // see: https://typescript-eslint.io/troubleshooting/faqs/eslint/#i-get-errors-from-the-no-undef-rule-about-global-variables-not-being-defined-even-though-there-are-no-typescript-errors
            "no-undef": "off",

            // Ignore unused parameters and variables that start with underscore
            "no-unused-vars": ["error", { argsIgnorePattern: "^_", varsIgnorePattern: "^_" }],
            "@typescript-eslint/no-unused-vars": [
                "error",
                { argsIgnorePattern: "^_", varsIgnorePattern: "^_" },
            ],
        },
    },
    {
        // Allow raw API helpers in the infrastructure layers
        files: [
            "src/lib/api/**/*.ts",
            "src/lib/providers/**/*.ts",
            "src/lib/utils/logger.ts",
            "src/test-setup.ts",
            "**/*.test.ts",
        ],
        rules: {
            "no-restricted-imports": "off",
            "no-console": "off",
        },
    },
    {
        files: ["**/*.svelte", "**/*.svelte.ts", "**/*.svelte.js"],
        languageOptions: {
            parserOptions: {
                projectService: true,
                extraFileExtensions: [".svelte"],
                parser: ts.parser,
                svelteConfig,
            },
        },
    },
);
