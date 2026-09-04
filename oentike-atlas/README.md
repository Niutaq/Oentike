# Oentike atlas (pilot pack)

Versioned knowledge cards for the field helper. **No scores here.** No stock photos, no grzyby.pl content, no photo-ID.

## Layout

| Path | Role |
|---|---|
| `schema/card.schema.json` | Shape of one species card |
| `packs/pilot-0.0.1/` | First unsigned pack (sign later with cosign/TUF) |
| `packs/pilot-0.0.1/pack.json` | Pack metadata + card list |
| `packs/pilot-0.0.1/cards/*.json` | One file per species |

## Pilot species

1. `boletus-edulis` — borowik szlachetny (scored in the conditions API)
2. `tylopilus-felleus` — goryczak żółciowy (lookalike, no score)
3. `boletus-reticulatus` — borowik usiatkowany (lookalike / cousin, no score)

Art stays out of git until we have our own plates (cream / ink / orange; **pores**, not gills).

## Owner checklist (learning beat)

Fill the empty scientific fields from a **cited** source (Mycobank / Index Fungorum for names; a named field guide for morphology). Do not invent edibility or protection status.

1. Open `cards/boletus-edulis.json`
2. Set `scientific_name`, PL/EN common names, `identifiers`, habitat, phenology, lookalikes, `protection_pl`, `citations`
3. Set `reviewed_at` to the day you checked the sources
4. Paste the filled JSON (or `git diff`) back into chat before we wire the UI
