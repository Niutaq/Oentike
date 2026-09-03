// @ts-check
import { defineConfig } from 'astro/config';

// https://astro.build/config
export default defineConfig({
    devToolbar: {
        enabled: false,
    },
    vite: {
        server: {
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
