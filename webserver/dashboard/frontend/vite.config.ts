import adapter from "@sveltejs/adapter-static";
import { vitePreprocess } from "@sveltejs/vite-plugin-svelte";
import devtoolsJson from "vite-plugin-devtools-json";
import { sveltekit } from "@sveltejs/kit/vite";
import { defineConfig } from "vitest/config";

export default defineConfig(({ mode }) => {
    return {
        server: {
            open: false,
            port: 5173, // Use default Vite port to avoid conflict with Go API
            proxy: {
                "/api": {
                    target: "http://localhost:8080", // The proxy API
                    changeOrigin: true,
                    secure: false,
                },
            },
            watch: {
                ignored: ["/var/", "/ssl/", "/build/", "/.svelte-kit/"],
            },
        },
        plugins: [
            sveltekit({
                preprocess: [vitePreprocess()],
                adapter: adapter({
                    fallback: "index.html",
                    precompress: false,
                    strict: true,
                }),
            }),
            devtoolsJson(),
        ],
        build: {
            sourcemap: mode === "development",
        },
        // Vitest runs in Node, so without this it would load Svelte's server build.
        resolve: process.env.VITEST ? { conditions: ["browser"] } : undefined,
        test: {
            environment: "jsdom",
            globals: true,
            include: ["src/**/*.{test,spec}.{js,ts}"],
            setupFiles: ["src/test-setup.ts"],
        },
    };
});
