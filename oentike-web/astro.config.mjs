// @ts-check
import { defineConfig } from 'astro/config';

// https://astro.build/config
export default defineConfig({
    devToolbar: {
        enabled: false,
    },
    vite: {
        // MapLibre resolves its ESM worker relative to import.meta.url.
        // Keep the module in its package directory during development.
        optimizeDeps: { exclude: ["maplibre-gl"] },
        server: {
            fs: {
                allow: [".."],
            },
            proxy: {
                "/api": {
                    target: "http://127.0.0.1:8081",
                    changeOrigin: true,
                    rewrite: (path) => path.replace(/^\/api/, ""),
                },
            },
        },
    },
});
