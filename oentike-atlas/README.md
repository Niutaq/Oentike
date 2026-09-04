# Oentike atlas (pilot pack)

Versioned knowledge cards for the field helper. **No scores here.** No stock photos, no grzyby.pl content, no photo-ID.

## Display rule

**Common name first, Latin beside/under it.** Polish UI → `names.pl`; English UI → `names.en`; `scientific_name` is secondary. Never lead with the slug or Latin alone.

## Layout

| Path | Role |
|---|---|
| `schema/card.schema.json` | Shape of one species card |
| `packs/pilot-0.0.1/` | First unsigned pack (sign later with cosign/TUF) |
| `packs/pilot-0.0.1/pack.json` | Pack metadata + card list |
| `packs/pilot-0.0.1/cards/*.json` | One file per species |

## Pilot species

1. Borowik szlachetny (`boletus-edulis`) — scored in the conditions API
2. Goryczak żółciowy (`tylopilus-felleus`) — lookalike, no score
3. Borowik usiatkowany (`boletus-reticulatus`) — lookalike / cousin, no score

Art stays out of git until we have our own plates (cream / ink / orange; **pores**, not gills).

## Owner checklist (learning beat)

Common names + Latin binomials for the pilot are already set. Fill the rest from a **cited** source (Mycobank / Index Fungorum for IDs; a named field guide for habitat / phenology / lookalike notes). Do not invent protection status.

1. Open `cards/boletus-edulis.json`
2. Set `identifiers`, `habitat`, `phenology`, lookalike notes, `protection_pl`, `citations`
3. Set `reviewed_at` to the day you checked the sources
4. Paste the filled JSON (or `git diff`) back into chat before we wire the UI
