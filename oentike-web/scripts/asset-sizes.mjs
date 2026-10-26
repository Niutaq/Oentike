import { readdir, readFile } from 'node:fs/promises';
import { gzipSync } from 'node:zlib';
import { fileURLToPath } from 'node:url';
const base = new URL('../dist/_astro/', import.meta.url);
const rows = [];
for (const file of await readdir(base)) {
  if (!/\.(js|css)$/.test(file)) continue;
  const data = await readFile(new URL(file, base));
  rows.push({ file, bytes: data.length, gzipBytes: gzipSync(data).length });
}
rows.sort((a,b) => b.bytes-a.bytes);
console.log(`Built assets: ${fileURLToPath(base)}`);
console.table(rows.map(({file,bytes,gzipBytes}) => ({file,KiB:(bytes/1024).toFixed(1),'gzip KiB':(gzipBytes/1024).toFixed(1)})));
console.log('Totals include shared chunks and all routes; these are not page-load, FPS or Tauri startup measurements.');
