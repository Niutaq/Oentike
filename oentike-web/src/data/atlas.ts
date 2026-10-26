import pack from "../../../oentike-atlas/packs/pilot-0.0.1/pack.json";

import { z } from "zod";

export const AtlasCardSchema = z.object({
    slug: z.string(),
    names: z.object({ pl: z.string(), en: z.string() }),
    scientific_name: z.string().nullable(),
    identifiers: z.object({
        mycobank: z.string().nullable(),
        index_fungorum: z.string().nullable(),
        gbif: z.string().nullable(),
    }),
    hymenophore: z.string(),
    description: z.string().nullable().optional(),
    habitat: z.string().nullable(),
    phenology: z.string().nullable(),
    lookalikes: z.array(z.object({
        slug: z.string(),
        note: z.string(),
        note_en: z.string(),
    })),
    text_en: z.object({
        description: z.string().nullable(),
        habitat: z.string().nullable(),
        phenology: z.string().nullable(),
        protection_pl: z.string().nullable(),
    }),
    protection_pl: z.string().nullable(),
    citations: z.array(z.object({ label: z.string(), url: z.string().nullable() })),
    art: z.object({
        cap: z.string().nullable(),
        hymenophore: z.string().nullable(),
        stem: z.string().nullable(),
        section: z.string().nullable(),
        lookalike_plate: z.string().nullable(),
    }),
    pack_version: z.string(),
    reviewed_at: z.string().nullable(),
});

export type AtlasCard = z.infer<typeof AtlasCardSchema>;

export const artViewOrder = [
    "cap",
    "hymenophore",
    "stem",
    "section",
    "lookalike_plate",
] as const;

export function cardArtViews(card: AtlasCard) {
    return artViewOrder
        .map((key) => {
            const src = card.art?.[key] ?? null;
            return src ? { key, src } : null;
        })
        .filter((view): view is { key: (typeof artViewOrder)[number]; src: string } =>
            Boolean(view),
        );
}

export const atlasPack = pack;

const modules = import.meta.glob<unknown>(
    "../../../oentike-atlas/packs/pilot-0.0.1/cards/*.json",
    { eager: true, import: "default" },
);

// The manifest is the single source of truth for both routes and validation.
export const atlasCards: AtlasCard[] = pack.cards.map((file) => {
    const card = modules[`../../../oentike-atlas/packs/pilot-0.0.1/${file}`];
    if (!card) throw new Error(`Atlas manifest references missing card: ${file}`);
    return AtlasCardSchema.parse(card);
}).sort((a, b) => a.names.pl.localeCompare(b.names.pl, "pl"));

export function getAtlasCard(slug: string): AtlasCard | undefined {
    return atlasCards.find((card) => card.slug === slug);
}

export function cardDisplayName(card: AtlasCard, lang: string): string {
    return lang.startsWith("en") ? card.names.en : card.names.pl;
}

export function lookalikeName(slug: string, lang: string): string {
    const card = getAtlasCard(slug);
    if (!card) return slug;
    return cardDisplayName(card, lang);
}
