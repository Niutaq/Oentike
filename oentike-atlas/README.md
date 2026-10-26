# Oentike Atlas

The working pack contains **13 species**. `packs/pilot-0.0.1/pack.json` is the source of truth for the cards displayed by the UI; adding a card to the manifest creates its page without a manual TypeScript import.

## Validation

Run this command from the repository root:

```sh
./oentike atlas
```

The validator uses JSON Schema 2020-12 and checks the manifest, versions, lookalike references, SVG availability, and consistency between the two artwork copies. It also runs before the UI build. Missing sources in older cards are reported as editorial debt, while structural errors block the build.

## Adding a species

1. Add a JSON file conforming to `schema/card.schema.json` under `packs/pilot-0.0.1/cards/`.
2. Add its path to `pack.json`. The Polish or English common name is primary and the Latin name is secondary.
3. Add a complete English version in `text_en` and `lookalikes[].note_en`, not only names and headings. The validator requires translation pairs. Write original, concise descriptions based on the cited sources. State the region covered by seasonal claims. Do not infer legal protection status from a source's silence.
4. Verify that every `lookalikes[].slug` points to an existing card. `folds` means chanterelle-like folds, `gills` means true gills, and `pores` means pores.
5. Supply four distinct anatomical SVGs: `SPECIES.cap.svg`, `SPECIES.stem.svg`, `SPECIES.section.svg`, and `SPECIES.hymenophore.svg`. Put originals under `atlas/art/v0/` and identical copies under `oentike-web/public/atlas/art/v0/`. Artwork paths are absolute UI asset paths. The validator blocks incomplete plates or reused views.
6. Run validation and the build. Inspect the rendered SVG; valid XML does not guarantee morphological accuracy.

## Editorial status

All 13 cards have four distinct anatomical plates, presented as cap, stem, longitudinal section and hymenophore. Six species have newly generated botanical engravings traced into real layered SVGs using the existing Boletus edulis artwork as a style reference; the seven existing four-plate sets remain in place. Four new cards (*Suillus granulatus*, *Hygrophoropsis aurantiaca*, *Chlorophyllum rhacodes*, and *Amanita phalloides*) have cited descriptions. Protection status and unverified identifiers are null. The artwork uses conventional colors and is not an identification key.

Seven older cards contain taxonomic links and pre-existing review dates; their descriptions and protection claims need a separate subject-matter review. The four illustrated cards and two migrated legacy cards have `reviewed_at: null`. A human reviewer sets `reviewed_at` after checking the content and citations; it must not be set automatically after fetching a page.

Source images, preserved whole-specimen portraits, anatomy references and vector-generation notes are in [atlas/source/v0](../atlas/source/v0/README.md). Each anatomical SVG is a separate drawing; color masks of a single portrait are not treated as anatomical views.

The pack is a working, unsigned artifact. Before publishing it, assign a new immutable version and agree on licensing for the content and artwork. Do not copy descriptions or photographs from external atlases.
