import edulis from "../../../oentike-atlas/packs/pilot-0.0.1/cards/boletus-edulis.json";
import reticulatus from "../../../oentike-atlas/packs/pilot-0.0.1/cards/boletus-reticulatus.json";
import felleus from "../../../oentike-atlas/packs/pilot-0.0.1/cards/tylopilus-felleus.json";
import cibarius from "../../../oentike-atlas/packs/pilot-0.0.1/cards/cantharellus-cibarius.json";
import procera from "../../../oentike-atlas/packs/pilot-0.0.1/cards/macrolepiota-procera.json";
import luteus from "../../../oentike-atlas/packs/pilot-0.0.1/cards/suillus-luteus.json";
import sulphureus from "../../../oentike-atlas/packs/pilot-0.0.1/cards/laetiporus-sulphureus.json";
import pack from "../../../oentike-atlas/packs/pilot-0.0.1/pack.json";

import { z } from "zod";

export const AtlasCardSchema = z.object({
    slug: z.string(),
    names: z.object({ pl: z.string(), en: z.string() }),
    scientific_name: z.string().nullable(),
    hymenophore: z.string(),
    habitat: z.string().nullable(),
    phenology: z.string().nullable(),
    lookalikes: z.array(z.object({ slug: z.string(), note: z.string() })),
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

export const atlasCards: AtlasCard[] = [
    AtlasCardSchema.parse(edulis),
    AtlasCardSchema.parse(felleus),
    AtlasCardSchema.parse(cibarius),
    AtlasCardSchema.parse(procera),
    AtlasCardSchema.parse(luteus),
    AtlasCardSchema.parse(sulphureus),
    AtlasCardSchema.parse(reticulatus),
];

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
