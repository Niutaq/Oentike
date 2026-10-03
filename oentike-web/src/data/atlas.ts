import pack from "../../../oentike-atlas/packs/pilot-0.0.1/pack.json";

export type AtlasCard = {
    slug: string;
    names: { pl: string; en: string };
    scientific_name: string | null;
    hymenophore: string;
    description?: string | null;
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

const modules = import.meta.glob<AtlasCard>(
    "../../../oentike-atlas/packs/pilot-0.0.1/cards/*.json",
    { eager: true, import: "default" },
);

// The manifest is the single source of truth for both routes and validation.
export const atlasCards: AtlasCard[] = pack.cards.map((file) => {
    const card = modules[`../../../oentike-atlas/packs/pilot-0.0.1/${file}`];
    if (!card) throw new Error(`Atlas manifest references missing card: ${file}`);
    return card;
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
