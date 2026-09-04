import edulis from "../../../oentike-atlas/packs/pilot-0.0.1/cards/boletus-edulis.json";
import reticulatus from "../../../oentike-atlas/packs/pilot-0.0.1/cards/boletus-reticulatus.json";
import felleus from "../../../oentike-atlas/packs/pilot-0.0.1/cards/tylopilus-felleus.json";
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
    pack_version: string;
    reviewed_at: string | null;
};

export const atlasPack = pack;

export const atlasCards: AtlasCard[] = [
    edulis as AtlasCard,
    felleus as AtlasCard,
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
