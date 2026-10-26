import { test, expect } from '@playwright/test';

test('native conditions and season requests use the chosen species without a borrowed score', async ({ page }) => {
 await page.route('**/api/readyz', route => route.fulfill({ json: { status: 'ready' } }));
 await page.addInitScript(() => {
  localStorage.setItem('oentike-language', 'pl');
  (window as any).speciesCalls = [];
  (window as any).__TAURI_INTERNALS__ = { invoke: async (command: string, args: any) => {
   (window as any).speciesCalls.push({command, args});
   if (command === 'get_conditions_ui') {
    const score = args.speciesSlug === 'boletus-edulis' ? 70 : 55;
    return { status: 'ready', score, factors: [], algorithmVersion: `oentike-conditions/0.1.0-${args.speciesSlug}` };
   }
   if (command === 'get_season_ui') return {days: []};
   return {cells: []};
  }};
 });
 await page.goto('/');
 await expect(page.locator('#score-value')).toHaveText('70');
 await page.locator('#species-select').selectOption('amanita-muscaria');
 await expect(page.locator('#score-value')).toHaveText('55');
 await expect(page.locator('#condition-title')).toContainText('Wstępna ocena');
 await expect.poll(() => page.evaluate(() => (window as any).speciesCalls.filter((c: any) => ['get_conditions_ui','get_season_ui'].includes(c.command) && c.args.speciesSlug === 'amanita-muscaria').length)).toBe(2);
});

test('species picker remembers selection, translates and links the correct card', async ({ page }) => {
 await page.addInitScript(() => localStorage.setItem('oentike-language', 'pl'));
 await page.goto('/');
 const picker = page.locator('#species-select');
 await expect(picker.locator('option')).toHaveCount(13);
 await picker.selectOption('amanita-phalloides');
 await expect(page.locator('#species-card-link')).toHaveAttribute('href', '/atlas/amanita-phalloides');
 await expect(page.locator('#species-model-note')).toContainText('Model wstępny');
 await page.reload();
 await expect(picker).toHaveValue('amanita-phalloides');
 await page.getByRole('button', { name: 'EN', exact: true }).click();
 await expect(picker.locator('option:checked')).toHaveText('Death cap');
 await expect(page.locator('#species-model-note')).toContainText('Provisional');
 await page.goto('/?species=cantharellus-cibarius');
 await expect(picker).toHaveValue('cantharellus-cibarius');
 await page.goto('/?species=invalid');
 await expect(picker).toHaveValue('boletus-edulis');
});

test('atlas classification has accessible text, sources and no ambiguous Amanita labels', async ({ page }) => {
 await page.addInitScript(() => localStorage.setItem('oentike-language', 'pl'));
 await page.goto('/atlas');
 await expect(page.locator('.edibility-badge')).toHaveCount(13);
 await expect(page.locator('.edibility-deadly')).toHaveText('Śmiertelnie trujący');
 await expect(page.locator('.edibility-poisonous')).toHaveText('Trujący');
 await page.goto('/atlas/amanita-phalloides');
 await expect(page.locator('.atlas-safety')).toContainText('Nie spożywaj');
 await expect(page.locator('.atlas-safety a')).toHaveAttribute('href', /gov.pl/);
 await page.locator('.atlas-conditions-link a').click();
 await expect(page.locator('#species-select')).toHaveValue('amanita-phalloides');
 await page.goto('/atlas/chlorophyllum-rhacodes');
 await expect(page.locator('.edibility-disputed')).toContainText('nie spożywaj');
 await page.getByRole('button', { name: 'EN', exact: true }).click();
 await expect(page.locator('.edibility-disputed')).toContainText('Disputed');
 await expect(page.locator('.atlas-safety')).toContainText('poisoning has been reported');
});

test('badges and species picker fit a narrow screen', async ({ page }, testInfo) => {
 await page.setViewportSize({ width: 390, height: 844 });
 await page.addInitScript(() => localStorage.setItem('oentike-language', 'pl'));
 await page.goto('/atlas');
 expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
 await page.screenshot({path: testInfo.outputPath('atlas-badges.png'), fullPage: true});
 await page.setViewportSize({ width: 1280, height: 1000 });
 await page.screenshot({path: testInfo.outputPath('atlas-badges-desktop.png'), fullPage: true});
 await page.evaluate(() => document.documentElement.dataset.theme = 'dark');
 await page.screenshot({path: testInfo.outputPath('atlas-badges-dark.png'), fullPage: true});
 await page.evaluate(() => document.documentElement.dataset.contrast = 'high');
 expect(await page.locator('.edibility-deadly').evaluate(el => getComputedStyle(el).color)).toBe('rgb(255, 255, 255)');
 await page.setViewportSize({ width: 390, height: 844 });
 await page.goto('/?species=chlorophyllum-rhacodes');
 await expect(page.locator('#species-select')).toHaveValue('chlorophyllum-rhacodes');
 expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
 await page.locator('.metadata').screenshot({path: testInfo.outputPath('species-picker.png')});
});
