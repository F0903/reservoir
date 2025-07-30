import devtoolsJson from "vite-plugin-devtools-json";
import { sveltekit } from "@sveltejs/kit/vite";
import { defineConfig } from "vite";

export default defineConfig({
    server: {
        port: 5173, // Use default Vite port to avoid conflict with Go API
        proxy: {
            "/api": {
                target: "http://localhost:8080", // Your Go API server
                changeOrigin: true,
                secure: false,
            },
        },
    },
    plugins: [sveltekit(), devtoolsJson()],
});
