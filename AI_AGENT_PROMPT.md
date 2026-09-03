# Oentike — Agent Context & Architecture Manifesto

**To the AI Agent reading this:** This file is the source of truth. Follow it before architectural decisions. The human (owner) is learning this stack on purpose: pair with them, do not silently finish every hard part. High-quality engineering, no AI-slop comments, no fake scores.

## 1. What Oentike is

Oentike is a **local field helper** for mushroom conditions (a modern, explainable take on the *idea* of grzyby.pl — not a scrape or a clone of their content). It answers: for a chosen species, in a chosen area, are conditions promising, and why?

It is also a **portfolio of real security / geospatial / cloud-native engineering** (readable to CBZC, SOC, cloud security): workload identity, signed data, cartography, offline, provenance. Niche technology is woven **into** the mushroom domain, not parked in a separate demo forever.

Inspiration from grzyby.pl’s **radar / mapa występowania** (how the *product* works, not their data):

- Coarse, regional, frequently updated picture of “how the season looks” — not a pin under a tree.
- Time dimension: last days + trend (their logged-in “9 days” / season histogram idea).
- Optional synoptic commentary later — generated from **our** factors, not copied text.
- Atlas as knowledge, linked to Mycobank / Index Fungorum / GBIF-style identifiers — our cards, our art, our citations.

Oentike stays **more local than the portal**: primary UX is one cell / one trip / one species (pilot: Lasy Janowskie near Targowisko/Studzieniec, `Boletus edulis`). The national/seasonal layer is **orientation**, not a crowdsourced treasure map.

## 2. Product layers (in order)

1. **Local conditions** — weather + forest context → versioned, low-confidence score for one H3 (or equivalent) cell. Missing data → explicit unavailable. Never invent a score.
2. **Seasonal awareness map** — choropleth of *our* condition potential (and later blurred observations) at województwo / H3 coarse resolution. Public layer never exposes exact GPS. Exact points stay private on-device.
3. **Offline field mode** — Tauri + local SQLite (encrypt private pins: SQLCipher or age). PMTiles / cell cache. Sync only over mTLS when the network returns.
4. **Built-in atlas** — versioned, signed pack. Hand-painted botanical art matching the UI (not stock photos, not grzyby.pl images). Scientific fields + citations. Lookalike pairs. Legal protection flags for PL. Pilot: porcini + 2–3 lookalikes.
5. **Gemma (last)** — local RAG over *our* atlas and score explanations. Never an edibility/identification verdict from a photo.

Crowd reports, if any, are opt-in, cell-blurred, and legally our own — not imported from grzyby.pl.

## 3. Engineering stack (use these in-domain)

| Role | Technology | Why |
|---|---|---|
| Workload identity | SPIFFE/SPIRE, X.509 SVIDs, mTLS | Ingest, scorer, API, Tauri/agent — no trust-on-network |
| Events | NATS JetStream | Ingest, score invalidation, sync (subjects about mushrooms/cells, not SOC) |
| Telemetry | OpenTelemetry → Jaeger, including store-and-forward offline | Provenance of the pipeline |
| Edge policy | Envoy (+ WASM when a real policy exists) | Authz from SPIFFE, limits — not a toy tollbooth on mushroom JSON without a rule |
| Geo | PostGIS, H3 (or S2), BDL via OGC API where possible, MapLibre, PMTiles | Cartography + offline tiles |
| Data trust | Signed packs (cosign and/or TUF): atlas, tiles, weather snapshots | Supply chain of *data* |
| Later geo intel | STAC + COG (e.g. Copernicus/Sentinel) | Habitat/moisture, not a second weather JSON |

Add SPIFFE, NATS, Envoy, OTel **onto the mushroom path** when a real ingest/sync need exists. Do not park a leftover SOC/FinOps/quarantine demo in this repo. Product data lives in `oentike-api` + PostGIS.

REST is a thin read API. Value is grid, signatures, offline, and explainable scores.

Skip Kubernetes/Istio/Cilium theater unless we actually operate a cluster.

## 4. Atlas rules

- Structured cards (JSON/SQLite): taxonomy IDs, PL/EN names, habitat, phenology, lookalikes, protection status, citation, pack version, reviewed_at.
- Art: one hand-painted system (cap, gills, stem, section, one lookalike). Brand-consistent.
- Gemma only after a real corpus exists; retrieval, not authority.

## 5. How the human learns (mandatory)

The owner wants to **do the work and learn**. The agent must not complete the whole stack unattended.

**Agent does:** scaffolding, wiring, reviews, explanations of *why*, small complete slices, checklists.

**Human does (agent stops and asks them to run / write):**

- Inspect a SPIRE SVID and a failing mTLS connection.
- Write or extend a PostGIS/H3 migration and query a cell.
- Fetch one BDL/OGC layer and say what CRS it is in.
- Sign and verify one atlas or snapshot pack.
- Sketch one atlas card (fields + one lookalike) from a cited source.
- Hit `task dev` and describe what came up.

When a task is a learning beat, say so, give the exact command or file, and wait for their result before proceeding. Teach in Polish if they write in Polish. No wall of generated config “as a gift”.

## 6. Quality rules

- Explain every score: inputs, freshness, algorithm version, confidence.
- Never fabricate conditions or observations.
- Protect exact locations on any shared/public surface.
- Idiomatic code per language; comments only for *why*. **No AI-slop comments.**
- Reproducible: Taskfile, Compose, mise/nix — no hidden machine snowflakes.

## 7. How to run

```bash
task dev          # PostGIS, API, Tauri conditions UI
```

Pilot slice remains: one area (Lasy Janowskie), one species. Open-Meteo ingest stores samples; `GetConditions` may fill factors but must not invent a score. Seasonal map and atlas packs come after the cell score is real — or in parallel only as empty, signed schemas, not fake heatmaps.
