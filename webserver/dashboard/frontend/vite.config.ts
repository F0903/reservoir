import devtoolsJson from "vite-plugin-devtools-json";
import { sveltekit } from "@sveltejs/kit/vite";
import { defineConfig } from "vite";

export default defineConfig({
    server: {
        open: true,
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
});
