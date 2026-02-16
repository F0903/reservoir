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
        plugins: [sveltekit(), devtoolsJson()],
        build: {
            sourcemap: mode === "development",
        },
        resolve: {
            conditions: ["import", "module", "browser", "svelte"],
        },
        test: {
            environment: "jsdom",
            globals: true,
            include: ["src/**/*.{test,spec}.{js,ts}"],
            setupFiles: ["src/test-setup.ts"],
        },
    };
});
