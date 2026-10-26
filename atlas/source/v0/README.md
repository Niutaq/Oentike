# Atlas artwork — 2026-10-04

Generated with the built-in `image_gen` tool, then traced with `tools/trace-atlas.py` (OpenCV palette quantisation and Potrace curves). Source PNGs preserve alpha. The SVGs have three colour layers: `form-light`, `accent`, `ink-shade`; these are colour masks, not anatomical segmentation. Original existing illustrations were not replaced. All four new images are schematic whole-specimen portraits and still require human mycological review.

Regenerate a vector from the saved PNG:

```sh
python3 tools/trace-atlas.py atlas/source/v0/SPECIES.png atlas/art/v0/SPECIES.cap.svg
```

Copy the SVG to the identical path under `oentike-web/public/` and run `./oentike atlas`.

## Prompts

### amanita-phalloides

Create a single original botanical illustration for the Oentike mushroom atlas: one whole Amanita phalloides (death cap) fruiting body, full stem base visible. Scientific educational, schematic morphological illustration, not a photograph. Elegant detailed linocut / screenprinted engraving: organic irregular contours, fine broken hatch marks, textured shadows, large cream negative-space highlights. Strict flat three-color palette: warm ivory #F6F5EF, near-black forest ink #19251E, burnt orange #E4772D as SPARING stylized shading; palette intentionally non-naturalistic. Match a sophisticated botanical woodcut, not a cartoon or geometric icon. Mushroom has broad smooth unspotted convex cap, pale free gills, tall slim stem with hanging skirt ring and conspicuous sack-like volva around bulbous base. No scales on cap, no snake-skin pattern. Centered isolated specimen occupying 75% of a square composition with generous margins. Fully transparent background; no ground, no shadows outside specimen, no labels, no letters, no text, no watermark. Make the anatomy clear, with enough irregular hand-engraved detail to vectorize into layered SVG.
### suillus-granulatus

Use case: scientific-educational. Create one original whole mushroom specimen portrait for Oentike Atlas. Subject: Suillus granulatus (weeping bolete). Low broad smooth convex cap, a finely dotted pale yellow PORES underside, absolutely NO gills, a medium stout cylindrical stem with tiny glandular dots toward the apex, absolutely NO ring, no volva. Match detailed botanical linocut / screenprint engraving, organic irregular contours, textured ink shadows, broken hatching, broad warm ivory highlights. Strict flat palette ivory #F6F5EF, forest black #19251E, burnt orange #E4772D for stylized accents; palette intentionally schematic. NOT photograph, NOT cartoon, NOT geometric icon. Center the complete specimen including full stem base in square canvas with 15% margins, detailed but vector-friendly. Transparent background, no landscape, no cast shadows outside mushroom, no text, no labels, no watermark. Anatomical silhouette is essential.

### hygrophoropsis-aurantiaca

Use case: scientific-educational. Create one original whole mushroom specimen portrait for Oentike Atlas. Subject: Hygrophoropsis aurantiaca (false chanterelle). Small depressed orange cap with gently rolled even edge, narrow densely spaced repeatedly FORKED true gills running far down a slender orange stem, no ring, no volva. Three-quarter view with underside clearly visible. Match detailed botanical linocut / screenprint engraving, organic irregular contours, textured ink shadows, broken hatching, broad warm ivory highlights. Strict flat palette ivory #F6F5EF, forest black #19251E, burnt orange #E4772D for stylized accents; palette intentionally schematic. NOT photograph, NOT cartoon, NOT geometric icon. Center the complete specimen including full stem base in square canvas with 15% margins, detailed but vector-friendly. Transparent background, no landscape, no cast shadows outside mushroom, no text, no labels, no watermark. Anatomical silhouette is essential.

### chlorophyllum-rhacodes

Use case: scientific-educational. Create one original whole mushroom specimen portrait for Oentike Atlas. Subject: Chlorophyllum rhacodes (shaggy parasol). Broad convex cap with conspicuous irregular shaggy scales, dark smooth central disc, pale free true gills, smooth unpatterned stem with prominent double ring and bulbous base, no sack-like volva and NO snakeskin pattern on stem. Match detailed botanical linocut / screenprint engraving, organic irregular contours, textured ink shadows, broken hatching, broad warm ivory highlights. Strict flat palette ivory #F6F5EF, forest black #19251E, burnt orange #E4772D for stylized accents; palette intentionally schematic. NOT photograph, NOT cartoon, NOT geometric icon. Center the complete specimen including full stem base in square canvas with 15% margins, detailed but vector-friendly. Transparent background, no landscape, no cast shadows outside mushroom, no text, no labels, no watermark. Anatomical silhouette is essential.
