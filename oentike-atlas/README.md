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
5. Put the original SVG under `atlas/art/v0/` at the repository root and an identical copy under `oentike-web/public/atlas/art/v0/`. Artwork paths are absolute UI asset paths.
6. Run validation and the build. Inspect the rendered SVG; valid XML does not guarantee morphological accuracy.

## Editorial status

Four new cards (*Suillus granulatus*, *Hygrophoropsis aurantiaca*, *Chlorophyllum rhacodes*, and *Amanita phalloides*) have cited descriptions and schematic portraits in the Oentike palette. Protection status and unverified identifiers are null. The artwork uses conventional colors and is not an identification key.

Seven older cards contain taxonomic links and pre-existing review dates; their descriptions and protection claims need a separate subject-matter review. The four illustrated cards and two migrated legacy cards have `reviewed_at: null`. A human reviewer sets `reviewed_at` after checking the content and citations; it must not be set automatically after fetching a page.

Source images, prompts, and vectorization notes for the four new portraits are in [atlas/source/v0](../atlas/source/v0/README.md). SVG layers represent colors; the portraits do not pretend to provide separate section or stem plates.

The pack is a working, unsigned artifact. Before publishing it, assign a new immutable version and agree on licensing for the content and artwork. Do not copy descriptions or photographs from external atlases.
