import { test, expect } from "@playwright/test";
import { readFileSync } from "node:fs";
const base = new URL("../../../oentike-atlas/packs/pilot-0.0.1/", import.meta.url);
const pack = JSON.parse(readFileSync(new URL("pack.json", base), "utf8"));

test("all atlas cards translate their body, comparisons and illustration descriptions", async ({ page }) => {
    const errors: string[] = [];
    page.on("pageerror", error => errors.push(error.message));
    await page.addInitScript(() => localStorage.setItem("oentike-language", "en"));
    for (const file of pack.cards) {
        const card = JSON.parse(readFileSync(new URL(file, base), "utf8"));
        await page.goto(`/atlas/${card.slug}/`);
        await expect(page.locator("h1")).toHaveText(card.names.en);
        await expect(page).toHaveTitle(`Oentike — ${card.names.en}`);
        for (const key of ["description", "habitat", "phenology", "protection_pl"]) {
            if (card[key]) await expect(page.locator(".atlas-body")).toContainText(card.text_en[key]);
        }
        for (const item of card.lookalikes) {
            await expect(page.locator(".atlas-lookalikes")).toContainText(item.note_en);
            expect(pack.cards).toContain(`cards/${item.slug}.json`);
        }
        const images = page.locator(".atlas-art-grid img");
        for (const img of await images.all()) {
            await expect(img).toHaveAttribute("alt", /schematic illustration/);
            await img.scrollIntoViewIfNeeded();
            await expect.poll(() => img.evaluate((el: HTMLImageElement) => el.complete && el.naturalWidth > 0)).toBe(true);
        }
        await page.getByRole("button", { name: "PL", exact: true }).click();
        await expect(page.locator("h1")).toHaveText(card.names.pl);
        if (card.habitat) await expect(page.locator(".atlas-body")).toContainText(card.habitat);
        await page.getByRole("button", { name: "EN", exact: true }).click();
    }
    expect(errors).toEqual([]);
});

test("compact area picker keeps readable text and fits a narrow screen", async ({ page }) => {
    await page.addInitScript(() => {
        localStorage.setItem("oentike-size", "small");
        localStorage.setItem("oentike-language", "pl");
    });
    await page.goto("/");
    await expect(page.locator("html")).toHaveAttribute("data-size", "small");
    expect(await page.locator(".metadata dt").first().evaluate(el => parseFloat(getComputedStyle(el).fontSize))).toBeGreaterThanOrEqual(12);
    expect(await page.locator("#area-search").evaluate(el => parseFloat(getComputedStyle(el).fontSize))).toBeGreaterThanOrEqual(14);
    await page.setViewportSize({ width: 390, height: 844 });
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
});


test("production map worker is bundled and executes", async ({ page }) => {
    await page.route("https://**/*", route => route.abort());
    const errors: string[] = [];
    page.on("pageerror", error => errors.push(error.message));
    const workerStarted = page.waitForEvent("worker");
    await page.goto("/");
    const worker = await workerStarted;
    const response = await page.request.get(worker.url());
    expect(response.ok()).toBe(true);
    expect(response.headers()["content-type"]).toMatch(/javascript/);
    await expect.poll(() => worker.evaluate(() => typeof (self as any).worker)).toBe("object");
    expect(errors).toEqual([]);
});
