import { defineConfig } from "@playwright/test";

export default defineConfig({
    testDir: "./tests/browser",
    use: { baseURL: "http://127.0.0.1:4322", locale: "pl-PL" },
    webServer: {
        command: "npm run preview -- --host 127.0.0.1 --port 4322",
        url: "http://127.0.0.1:4322/expedition/",
        reuseExistingServer: false,
    },
});
