import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
    testDir: ".",
    testMatch: ["e2e/**/*.spec.ts", "tests/browser/**/*.spec.ts"],
    fullyParallel: true,
    forbidOnly: Boolean(process.env.CI),
    retries: process.env.CI ? 2 : 0,
    workers: process.env.CI ? 1 : undefined,
    reporter: "html",
    projects: [
        { name: "chromium", use: { ...devices["Desktop Chrome"] } },
        { name: "webkit", use: { ...devices["Desktop Safari"] } },
    ],
    use: {
        baseURL: "http://127.0.0.1:4322",
        locale: "pl-PL",
        trace: "on-first-retry",
    },
    webServer: {
        command: "npm run build && npm run preview -- --host 127.0.0.1 --port 4322",
        url: "http://127.0.0.1:4322/",
        reuseExistingServer: !process.env.CI,
    },
});
