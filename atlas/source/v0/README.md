# Atlas artwork

## Anatomical plates — 2026-10-08

Every card now contains four separate SVG views: cap exterior, entire stem with base, longitudinal section, and underside/hymenophore. The six completed sets use original botanical engravings generated with the built-in `image_gen` tool, using the existing Boletus edulis illustrations as the visual quality reference (`style-reference.png`). All 24 RGBA source images are saved under `anatomical/`; the complete prompt set and generation method are recorded in `anatomical-prompts.json`. Each image was traced with `tools/trace-atlas.py --size 1536 --detail`, which preserves fine hatch strokes, avoids median smoothing and uses tighter Potrace curve tolerances. The resulting SVGs contain actual paths and three color layers, without embedded rasters, fonts, scripts or external references. The remaining seven existing complete sets are preserved.

The four earlier whole-specimen SVGs are preserved here as `SPECIES.portrait.svg` alongside their source PNGs. Their `cap.svg` paths now serve newly illustrated cap-focused portraits in the style of the existing bolete plates. Each stem, section and hymenophore plate is separately generated; these are not crops or color masks of the cap portrait. Old portrait PNGs must not be regenerated over the new anatomical plates.

Anatomical references and drawing choices (descriptions consulted; no tracing of source photographs):

- [Amanita muscaria — Tom Volk, University of Wisconsin](https://botit.botany.wisc.edu/toms_fungi/amusc.html): cap with veil warts, free gills, skirt ring and bulb with concentric veil remnants; no sack-like volva.
- [Amanita phalloides — Michael Kuo](https://www.mushroomexpert.com/amanita_phalloides.html): smooth cap, free close gills, skirt ring and sack-like volva; white unchanging cut flesh.
- [Chlorophyllum rhacodes — First Nature](https://www.first-nature.com/fungi/chlorophyllum-rhacodes.php): shaggy cap scales, free gills, double ring, bulbous smooth stem without snakeskin markings; orange accent at cut surfaces represents reddening.
- [Hygrophoropsis aurantiaca — Michael Kuo](https://www.mushroomexpert.com/hygrophoropsis_aurantiaca.html): shallowly depressed cap, fine repeatedly forked decurrent true gills, slender ringless stem and thin cut flesh.
- [Suillus granulatus — Michael Kuo](https://www.mushroomexpert.com/suillus_granulatus.html): smooth sticky cap, small pores with occasional milky droplets, short tubes, glandular dots near the apex of a ringless stem.
- [Imleria badia — Michael Kuo](https://www.mushroomexpert.com/imleria_badia.html): smooth convex cap, small pores and tubes, longitudinally wrinkled stem without a reticulate net or ring.

Print colors are deliberately schematic. Shapes and views still require human mycological review; they are not an identification key and no review date is set by generation.

Regenerate one anatomical SVG from its saved source:

```sh
python3 tools/trace-atlas.py atlas/source/v0/anatomical/amanita-phalloides.cap.png atlas/art/v0/amanita-phalloides.cap.svg --size 1536 --detail
cp atlas/art/v0/amanita-phalloides.cap.svg oentike-web/public/atlas/art/v0/amanita-phalloides.cap.svg
./oentike atlas
```

Render a comparison sheet from actual SVGs (requires installed Playwright Chromium):

```sh
cd oentike-web && node scripts/preview-atlas.mjs
```

## Archived portrait sources — 2026-10-04

Generated with the built-in `image_gen` tool, then traced with `tools/trace-atlas.py` (OpenCV palette quantisation and Potrace curves). Source PNGs preserve alpha. The SVGs have three colour layers: `form-light`, `accent`, `ink-shade`; these are colour masks, not anatomical segmentation. Original existing illustrations were not replaced. All four new images are schematic whole-specimen portraits and still require human mycological review.

Regenerate a vector from the saved PNG:

```sh
python3 tools/trace-atlas.py atlas/source/v0/SPECIES.png /tmp/SPECIES.portrait.svg
```

The PNGs regenerate whole-specimen portraits only. Do not use these as replacements for the four anatomical views in the cards.

## Prompts

### amanita-phalloides

Create a single original botanical illustration for the Oentike mushroom atlas: one whole Amanita phalloides (death cap) fruiting body, full stem base visible. Scientific educational, schematic morphological illustration, not a photograph. Elegant detailed linocut / screenprinted engraving: organic irregular contours, fine broken hatch marks, textured shadows, large cream negative-space highlights. Strict flat three-color palette: warm ivory #F6F5EF, near-black forest ink #19251E, burnt orange #E4772D as SPARING stylized shading; palette intentionally non-naturalistic. Match a sophisticated botanical woodcut, not a cartoon or geometric icon. Mushroom has broad smooth unspotted convex cap, pale free gills, tall slim stem with hanging skirt ring and conspicuous sack-like volva around bulbous base. No scales on cap, no snake-skin pattern. Centered isolated specimen occupying 75% of a square composition with generous margins. Fully transparent background; no ground, no shadows outside specimen, no labels, no letters, no text, no watermark. Make the anatomy clear, with enough irregular hand-engraved detail to vectorize into layered SVG.
### suillus-granulatus

Use case: scientific-educational. Create one original whole mushroom specimen portrait for Oentike Atlas. Subject: Suillus granulatus (weeping bolete). Low broad smooth convex cap, a finely dotted pale yellow PORES underside, absolutely NO gills, a medium stout cylindrical stem with tiny glandular dots toward the apex, absolutely NO ring, no volva. Match detailed botanical linocut / screenprint engraving, organic irregular contours, textured ink shadows, broken hatching, broad warm ivory highlights. Strict flat palette ivory #F6F5EF, forest black #19251E, burnt orange #E4772D for stylized accents; palette intentionally schematic. NOT photograph, NOT cartoon, NOT geometric icon. Center the complete specimen including full stem base in square canvas with 15% margins, detailed but vector-friendly. Transparent background, no landscape, no cast shadows outside mushroom, no text, no labels, no watermark. Anatomical silhouette is essential.

### hygrophoropsis-aurantiaca

Use case: scientific-educational. Create one original whole mushroom specimen portrait for Oentike Atlas. Subject: Hygrophoropsis aurantiaca (false chanterelle). Small depressed orange cap with gently rolled even edge, narrow densely spaced repeatedly FORKED true gills running far down a slender orange stem, no ring, no volva. Three-quarter view with underside clearly visible. Match detailed botanical linocut / screenprint engraving, organic irregular contours, textured ink shadows, broken hatching, broad warm ivory highlights. Strict flat palette ivory #F6F5EF, forest black #19251E, burnt orange #E4772D for stylized accents; palette intentionally schematic. NOT photograph, NOT cartoon, NOT geometric icon. Center the complete specimen including full stem base in square canvas with 15% margins, detailed but vector-friendly. Transparent background, no landscape, no cast shadows outside mushroom, no text, no labels, no watermark. Anatomical silhouette is essential.

### chlorophyllum-rhacodes

Use case: scientific-educational. Create one original whole mushroom specimen portrait for Oentike Atlas. Subject: Chlorophyllum rhacodes (shaggy parasol). Broad convex cap with conspicuous irregular shaggy scales, dark smooth central disc, pale free true gills, smooth unpatterned stem with prominent double ring and bulbous base, no sack-like volva and NO snakeskin pattern on stem. Match detailed botanical linocut / screenprint engraving, organic irregular contours, textured ink shadows, broken hatching, broad warm ivory highlights. Strict flat palette ivory #F6F5EF, forest black #19251E, burnt orange #E4772D for stylized accents; palette intentionally schematic. NOT photograph, NOT cartoon, NOT geometric icon. Center the complete specimen including full stem base in square canvas with 15% margins, detailed but vector-friendly. Transparent background, no landscape, no cast shadows outside mushroom, no text, no labels, no watermark. Anatomical silhouette is essential.
