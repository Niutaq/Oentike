import edulis from "../../../oentike-atlas/packs/pilot-0.0.1/cards/boletus-edulis.json";
import reticulatus from "../../../oentike-atlas/packs/pilot-0.0.1/cards/boletus-reticulatus.json";
import felleus from "../../../oentike-atlas/packs/pilot-0.0.1/cards/tylopilus-felleus.json";
import cibarius from "../../../oentike-atlas/packs/pilot-0.0.1/cards/cantharellus-cibarius.json";
import procera from "../../../oentike-atlas/packs/pilot-0.0.1/cards/macrolepiota-procera.json";
import luteus from "../../../oentike-atlas/packs/pilot-0.0.1/cards/suillus-luteus.json";
import sulphureus from "../../../oentike-atlas/packs/pilot-0.0.1/cards/laetiporus-sulphureus.json";
import pack from "../../../oentike-atlas/packs/pilot-0.0.1/pack.json";

export type AtlasCard = {
    slug: string;
    names: { pl: string; en: string };
    scientific_name: string | null;
    hymenophore: string;
    habitat: string | null;
    phenology: string | null;
    lookalikes: { slug: string; note: string }[];
    protection_pl: string | null;
    citations: { label: string; url: string | null }[];
    art: {
        cap: string | null;
        hymenophore: string | null;
        stem: string | null;
        section: string | null;
        lookalike_plate: string | null;
    };
    pack_version: string;
    reviewed_at: string | null;
};

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
    edulis as AtlasCard,
    felleus as AtlasCard,
    cibarius as AtlasCard,
    procera as AtlasCard,
    luteus as AtlasCard,
    sulphureus as AtlasCard,
    reticulatus as AtlasCard,
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
