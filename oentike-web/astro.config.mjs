// @ts-check
import { defineConfig } from 'astro/config';

// https://astro.build/config
export default defineConfig({
    devToolbar: {
        enabled: false,
    },
    // Bind IPv4 explicitly so health checks and the macOS .app WebView
    // (http://127.0.0.1:4321) hit the same listener as curl — not only ::1.
    server: {
        host: "127.0.0.1",
        port: 4321,
    },
    vite: {
        // MapLibre resolves its ESM worker relative to import.meta.url.
        // Keep the module in its package directory during development.
        optimizeDeps: { exclude: ["maplibre-gl"] },
        server: {
            host: "127.0.0.1",
            strictPort: true,
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

