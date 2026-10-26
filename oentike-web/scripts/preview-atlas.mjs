import { chromium } from '@playwright/test';
import { readFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';

const root = new URL('../../', import.meta.url);
const species = ['boletus-edulis', 'amanita-phalloides', 'amanita-muscaria', 'chlorophyllum-rhacodes', 'hygrophoropsis-aurantiaca', 'suillus-granulatus', 'imleria-badia'];
const views = ['cap', 'stem', 'section', 'hymenophore'];
const rows = await Promise.all(species.map(async slug => {
    const images = await Promise.all(views.map(async view => {
        const svg = await readFile(new URL(`atlas/art/v0/${slug}.${view}.svg`, root));
        return `<figure><img src="data:image/svg+xml;base64,${svg.toString('base64')}"/><figcaption>${view}</figcaption></figure>`;
    }));
    return `<section><h2>${slug}${slug === 'boletus-edulis' ? ' · style reference' : ''}</h2><div>${images.join('')}</div></section>`;
}));
const browser = await chromium.launch();
try {
    const page = await browser.newPage({ viewport: { width: 1280, height: 1000 }, deviceScaleFactor: 1 });
    await page.setContent(`<html><head><style>
        body{margin:0;padding:20px;background:#eceee6;color:#19251e;font:16px system-ui}
        h2{margin:12px 0 0;font-size:18px;font-weight:500} section>div{display:flex}
        figure{width:310px;margin:0;text-align:center} img{width:300px;height:300px;object-fit:contain}
        figcaption{font-size:14px;padding-bottom:12px}
    </style></head><body>${rows.join('')}</body></html>`);
    await page.locator('img').evaluateAll(images => Promise.all(images.map(image => image.decode())));
    const destination = fileURLToPath(new URL('atlas/source/v0/anatomical-preview.png', root));
    await page.screenshot({ path: destination, fullPage: true });
    console.log(destination);
} finally { await browser.close(); }
