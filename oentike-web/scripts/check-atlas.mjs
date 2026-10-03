import { readFile, readdir } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
import Ajv2020 from 'ajv/dist/2020.js';

const root = fileURLToPath(new URL('../../', import.meta.url));
const base = path.join(root, 'oentike-atlas/packs/pilot-0.0.1');
const json = async (file) => JSON.parse(await readFile(file, 'utf8'));
const schema = await json(path.join(root, 'oentike-atlas/schema/card.schema.json'));
const validate = new Ajv2020({ allErrors: true }).compile(schema);
const pack = await json(path.join(base, 'pack.json'));
const errors = [];
const warnings = [];
const cards = new Map();
let assets = 0;
let artBytes = 0;
const seenAssets = new Set();
if (!Array.isArray(pack.cards) || !pack.cards.length) throw new Error('Empty atlas manifest');
if (new Set(pack.cards).size !== pack.cards.length) errors.push('Duplicate manifest entries');
for (const file of pack.cards) {
  if (!/^cards\/[a-z0-9]+(?:-[a-z0-9]+)*\.json$/.test(file)) { errors.push(`Invalid card path: ${file}`); continue; }
  let card;
  try { card = await json(path.join(base, file)); } catch (error) { errors.push(`${file}: ${error.message}`); continue; }
  if (!validate(card)) { errors.push(`${file}: ${JSON.stringify(validate.errors)}`); continue; }
  if (file !== `cards/${card.slug}.json`) errors.push(`${file}: slug differs from filename`);
  if (cards.has(card.slug)) errors.push(`Duplicate slug: ${card.slug}`);
  cards.set(card.slug, card);
  if (card.pack_version !== pack.version) errors.push(`${file}: pack version mismatch`);
  if (!card.citations.length) warnings.push(`${card.slug}: no sources yet`);
  if (card.reviewed_at && (!/^\d{4}-\d{2}-\d{2}$/.test(card.reviewed_at) || Number.isNaN(Date.parse(card.reviewed_at)))) errors.push(`${file}: invalid review date`);
  for (const citation of card.citations) {
    if (citation.url && !/^https?:\/\//.test(citation.url)) errors.push(`${file}: invalid source URL`);
  }
  for (const src of Object.values(card.art ?? {}).filter(Boolean)) {
    if (!/^\/atlas\/art\/v\d+\/[a-z0-9.-]+\.svg$/.test(src)) { errors.push(`${file}: invalid art path ${src}`); continue; }
    if (seenAssets.has(src)) continue;
    seenAssets.add(src);
    try {
      const source = await readFile(path.join(root, src.slice(1)));
      const served = await readFile(path.join(root, 'oentike-web/public', src.slice(1)));
      if (!source.equals(served)) errors.push(`${src}: source and public copy differ`);
      const svg = source.toString('utf8');
      if (!/<svg\b/.test(svg) || !/viewBox=/.test(svg) || /<(?:script|foreignObject)\b|\bon\w+\s*=|(?:href|xlink:href)\s*=\s*["'](?!#)|<!DOCTYPE|<!ENTITY/i.test(svg)) errors.push(`${src}: SVG must be self-contained and passive`);
      assets++; artBytes += served.length;
    } catch (error) { errors.push(`${src}: ${error.message}`); }
  }
}
for (const file of await readdir(path.join(base, 'cards'))) {
  if (file.endsWith('.json') && !pack.cards.includes(`cards/${file}`)) errors.push(`Unlisted card: ${file}`);
}
for (const card of cards.values()) {
  for (const other of card.lookalikes) {
    if (!cards.has(other.slug)) errors.push(`${card.slug}: missing lookalike ${other.slug}`);
    if (other.slug === card.slug) errors.push(`${card.slug}: self-reference`);
  }
}
for (const warning of warnings) console.warn(`REVIEW: ${warning}`);
for (const error of errors) console.error(`ERROR: ${error}`);
console.log(`Atlas: ${cards.size} cards; ${assets} SVGs; ${(artBytes/1024).toFixed(1)} KiB; ${errors.length} errors; ${[...cards.values()].filter(c => !c.reviewed_at).length} cards awaiting human review.`);
if (errors.length) process.exitCode = 1;
