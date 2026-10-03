import { test, expect } from "@playwright/test";

test.beforeEach(async ({ page }) => {
    await page.addInitScript(() => {
        localStorage.setItem("oentike-language", "pl");
        let callback: PositionCallback | null = null;
        Object.defineProperty(navigator, "geolocation", { value: {
            watchPosition(success: PositionCallback) { callback = success; return 1; },
            clearWatch() { callback = null; },
            getCurrentPosition(_success: PositionCallback, failure: PositionErrorCallback) {
                failure({ code: 1 } as GeolocationPositionError);
            },
        } });
        (window as any).emitFix = (lat: number, time: number, accuracy = 8) => callback?.({
            timestamp: time, coords: { latitude: lat, longitude: 22, accuracy },
        } as GeolocationPosition);
    });
});

test("journal works without API, restores after reload and exports disconnected segments", async ({ page }) => {
    const errors: string[] = [];
    const forbidden: string[] = [];
    page.on("pageerror", error => errors.push(error.message));
    await page.route("**/*", route => {
        const url = new URL(route.request().url());
        if (url.origin !== "http://127.0.0.1:4322" || url.pathname.startsWith("/api")) {
            forbidden.push(url.href);
            return route.abort();
        }
        return route.continue();
    });
    await page.goto("/expedition/");
    await expect(page.locator("#trip-fields")).toBeEnabled();
    await page.locator("#trip-name").fill("Janów — spacer");
    await page.locator("#parking-lat").fill("50");
    await page.locator("#parking-lon").fill("22");
    await page.locator("button[type=submit]").click();
    await expect(page.locator("#active-trip")).toHaveText("Janów — spacer");
    await page.locator("#record").click();
    const start = Date.now() - 20000;
    await page.evaluate(time => (window as any).emitFix(50, time), start);
    await expect(page.locator("#sample-count")).toHaveText("1");
    await page.evaluate(time => (window as any).emitFix(50.001, time), start + 6000);
    await expect(page.locator("#sample-count")).toHaveText("2");
    await expect(page.locator("#trail-distance")).toHaveText("111 m");
    await page.reload();
    await expect(page.locator("#sample-count")).toHaveText("2");
    await expect(page.locator("#pause")).toBeDisabled();
    await page.locator("#record").click();
    await page.evaluate(() => (window as any).emitFix(50.01, Date.now()));
    await expect(page.locator("#sample-count")).toHaveText("3");
    await expect(page.locator("#trail-distance")).toHaveText("111 m");
    await page.locator("#pause").click();
    const download = page.waitForEvent("download");
    await page.locator("#export-trip").click();
    const file = await download;
    const stream = await file.createReadStream();
    const chunks = [];
    for await (const chunk of stream!) chunks.push(chunk);
    const data = JSON.parse(Buffer.concat(chunks).toString());
    expect(data.features.map((f: any) => f.geometry.type)).toEqual(["Point", "LineString", "Point"]);
    expect(data.features[0].properties.accuracy).toBeNull();
    await page.getByRole("button", { name: "EN", exact: true }).click();
    await expect(page.locator("#record")).toHaveText("Record trail");
    await expect(page).toHaveTitle("Oentike — Expedition");
    expect(forbidden).toEqual([]);
    expect(errors).toEqual([]);
});

test("location denial and inaccurate fixes are visible and never create samples", async ({ page }) => {
    await page.goto("/expedition/");
    await page.locator("#locate-parking").click();
    await expect(page.locator("#field-status")).toContainText("Brak zgody");
    await page.locator("#trip-name").fill("Test");
    await page.locator("#parking-lat").fill("50");
    await page.locator("#parking-lon").fill("22");
    await page.locator("button[type=submit]").click();
    await page.locator("#record").click();
    await page.evaluate(() => (window as any).emitFix(50, Date.now(), 200));
    await expect(page.locator("#field-status")).toContainText("Pomiar pominięty");
    await expect(page.locator("#sample-count")).toHaveText("0");
    await page.setViewportSize({ width: 390, height: 844 });
    await expect(page.getByRole("link", { name: "Wyprawa", exact: true })).toBeVisible();
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
});
