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
    await expect(page.locator("#location-settings")).toBeHidden();
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


test("save and edit a plan without GPS, then attach parking to the same expedition", async ({ page }) => {
    await page.goto("/expedition/");
    await page.locator("#trip-name").fill("Sobota w lesie");
    await page.locator("#trip-area").fill("Lasy Janowskie");
    await page.locator("#trip-date").fill("2026-10-10");
    await page.locator("#trip-notes").fill("Termos i koszyk");
    await page.locator("#save-trip").click();
    await expect(page.locator("#field-status")).toContainText("Plan zapisany");
    await expect(page.locator("#record")).toBeDisabled();
    await page.reload();
    await expect(page.locator("#active-trip")).toHaveText("Sobota w lesie");
    await expect(page.locator("#trip-summary")).toContainText("Lasy Janowskie · 2026-10-10");
    await expect(page.locator("#trip-note-preview")).toHaveText("Termos i koszyk");
    await page.locator("#edit-trip").click();
    await page.locator("#parking-lat").fill("50.7");
    await page.locator("#parking-lon").fill("22.4");
    await page.locator("#trip-notes").fill("Termos, koszyk i mapa");
    await page.locator("#save-trip").click();
    await expect(page.locator("#field-status")).toContainText("Zmiany zapisane");
    await expect(page.locator("#record")).toBeEnabled();
    await expect(page.locator("#trip-select option")).toHaveCount(1);
    await page.reload();
    await expect(page.locator("#parking-coordinates")).toContainText("50.700000, 22.400000");
    await expect(page.locator("#trip-note-preview")).toHaveText("Termos, koszyk i mapa");
    await page.getByRole("button", { name: "EN", exact: true }).click();
    await expect(page.locator("#edit-trip")).toHaveText("Edit plan / add parking");
});

test("macOS denial offers settings only on user action and location can be retried", async ({ page }) => {
    await page.addInitScript(() => {
        Object.defineProperty(navigator, "userAgent", { value: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)" });
        (window as any).isTauri = true;
        (window as any).settingsCalls = [];
        (window as any).locationPurposes = [];
        (window as any).locationAllowed = false;
        (window as any).__TAURI_INTERNALS__ = {
            invoke: async (command: string, args?: { purpose?: string }) => {
                (window as any).settingsCalls.push(command);
                if (command === 'get_current_location') {
                    (window as any).locationPurposes.push(args?.purpose);
                    if (!(window as any).locationAllowed) throw { code: 1 };
                    return { timestamp: Date.now(), latitude: 50.7, longitude: 22.4, accuracy: 8 };
                }
            },
        };
        navigator.geolocation.getCurrentPosition = () => { throw new Error('Desktop must use native location'); };
    });
    await page.goto("/expedition/");
    await expect(page.locator("#location-settings")).toBeHidden();
    await page.locator("#locate-parking").click();
    await expect(page.locator("#field-status")).toContainText("Usługi lokalizacji");
    await expect(page.locator("#location-settings")).toBeVisible();
    expect(await page.evaluate(() => (window as any).settingsCalls)).toEqual(["get_current_location"]);
    expect(await page.evaluate(() => (window as any).locationPurposes)).toEqual(["parking"]);
    await page.locator("#location-settings").click();
    expect(await page.evaluate(() => (window as any).settingsCalls)).toEqual(["get_current_location", "open_location_settings"]);
    await expect(page.locator("#field-status")).toContainText("wróć do wyprawy");
    await page.evaluate(() => {
        (window as any).locationAllowed = true;
    });
    await page.locator("#locate-parking").click();
    await expect(page.locator("#parking-lat")).toHaveValue("50.7");
    await expect(page.locator("#parking-lon")).toHaveValue("22.4");
    await expect(page.locator("#location-settings")).toBeHidden();
});

test("macOS untrusted host fails closed without opening location settings", async ({ page }) => {
    await page.addInitScript(() => {
        Object.defineProperty(navigator, "userAgent", { value: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)" });
        (window as any).__TAURI_INTERNALS__ = {
            invoke: async (command: string) => {
                if (command === "get_current_location") throw { code: 4 };
            },
        };
        navigator.geolocation.getCurrentPosition = () => { throw new Error("Desktop must use native location"); };
    });
    await page.goto("/expedition/");
    await page.locator("#locate-parking").click();
    await expect(page.locator("#field-status")).toContainText("./oentike field");
    await expect(page.locator("#location-settings")).toBeHidden();
});
