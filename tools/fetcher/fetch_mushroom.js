import fs from 'fs/promises';
import path from 'path';

const args = process.argv.slice(2);
if (args.length === 0) {
    console.error("Usage: node fetch_mushroom.js 'Amanita muscaria'");
    process.exit(1);
}

const mushroomName = args.join(' ');
const slug = mushroomName.toLowerCase().replace(/\s+/g, '-');

async function fetchGBIFData(name) {
    const response = await fetch(`https://api.gbif.org/v1/species/match?name=${encodeURIComponent(name)}`);
    if (!response.ok) throw new Error(`GBIF API Error: ${response.status}`);
    return await response.json();
}

async function main() {
    try {
        const gbifData = await fetchGBIFData(mushroomName);

        const cardTemplate = {
            slug: slug,
            names: {
                pl: "TODO",
                en: gbifData.vernacularName || "TODO"
            },
            scientific_name: gbifData.scientificName || mushroomName,
            hymenophore: "TODO",
            habitat: null,
            phenology: null,
            lookalikes: [],
            protection_pl: null,
            citations: [
                {
                    label: "GBIF Backbone Taxonomy",
                    url: gbifData.usageKey ? `https://www.gbif.org/species/${gbifData.usageKey}` : null
                }
            ],
            art: {
                cap: null,
                hymenophore: null,
                stem: null,
                section: null,
                lookalike_plate: null
            },
            pack_version: "pilot-0.0.1",
            reviewed_at: new Date().toISOString()
        };

        const outDir = path.resolve(process.cwd(), '../../oentike-atlas/packs/pilot-0.0.1/cards');
        const outFile = path.join(outDir, `${slug}.json`);

        await fs.writeFile(outFile, JSON.stringify(cardTemplate, null, 4), 'utf-8');
        
        console.log(`Success: ${outFile}`);

    } catch (error) {
        console.error("Error:", error.message);
    }
}

main();
