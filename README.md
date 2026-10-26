<div align="center">

<img src="./oentike-web/public/oentike-wordmark.svg" alt="Oentike" width="192" />

<h1>Oentike</h1>

Explainable mushroom-condition scores for Polish forests.
Desktop app with a versioned species atlas, season trend chart, and offline expedition journal — backed by weather data, not crowd opinion.

</div>

---

## Table of Contents

- [Problem Statement](#problem-statement)
- [Architecture & Tech Stack](#architecture--tech-stack)
- [Data Integrity](#data-integrity)
- [Quick Start](#quick-start)
- [Known Limitations](#known-limitations)
- [Roadmap](#roadmap)
- [Security & Contributing](#security--contributing)
- [License](#license)
- [Screenshots](#screenshots)

---

## Problem Statement

Polish mushroom sites show crowdsourced pins ("found boletes here!") or vague national forecasts. Neither explains **why** conditions are good, neither works offline in the forest, and exact GPS pins on public maps erode foraging spots.

**Oentike solves this by:**

- **Explainable scoring** — every condition score (0–100) shows the weather factors behind it (precipitation, soil temperature, soil moisture), the algorithm version, and a confidence level. No black boxes, no invented numbers.
- **Per-species models** — each of the 13 pilot species has its own environmental envelope. Unknown slugs stay unscored rather than borrowing another taxon's result.
- **Searchable BDL nadleśnictwa** — ~429 Polish forest districts pulled from GUGiK WFS, each materialized as a spatial cell with its own weather ingest.
- **Offline expedition journal** — plan parking → trail → waypoints on-device (IndexedDB + GeoJSON). Private GPS stays private; nothing leaves the machine.
- **Versioned botanical atlas** — 13 species cards with hand-drawn anatomical SVG plates, lookalike cross-references, and cited taxonomy. Knowledge only — no edibility verdicts, no photo ID.

---

## Architecture & Tech Stack

| Component | Technology | Role |
|---|---|---|
| API | [![Go 1.27](https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white)](./oentike-api) · gRPC + HTTP health | Conditions scoring, season history, cell search, Open-Meteo ingest, BDL sync |
| Database | [![PostGIS 17](https://img.shields.io/badge/PostGIS-17--3.5-336791?logo=postgresql&logoColor=white)](./docker-compose.yml) | Spatial cells (EPSG:2180), weather samples, condition scores, forest units |
| Proto | [![Protobuf](https://img.shields.io/badge/Protobuf-v3-4285F4?logo=google&logoColor=white)](./oentike-proto/conditions.proto) | `ConditionsService` contract: `GetConditions`, `GetSeason`, `SearchCells`, `EnsureCell` |
| Desktop | [![Tauri 2](https://img.shields.io/badge/Tauri-2.11-FFC131?logo=tauri&logoColor=white)](./oentike-web/src-tauri) · Rust 1.97 | Native shell with Core Location (macOS), ed25519 pack verification, gRPC bridge |
| UI | [![Astro 7](https://img.shields.io/badge/Astro-7.1-BC52EE?logo=astro&logoColor=white)](./oentike-web) · MapLibre · GSAP | Conditions dashboard, interactive map, expedition planner, species atlas pages |
| Atlas | [JSON Schema 2020-12](./oentike-atlas/schema/card.schema.json) | 13 versioned species cards + 4 anatomical SVGs each; validated before every build |
| Observability | [![OpenTelemetry](https://img.shields.io/badge/OTel-Jaeger-5C2D91?logo=opentelemetry&logoColor=white)](./docker-compose.yml) | Optional distributed traces (`OTEL_EXPORTER_OTLP_ENDPOINT`); off by default |
| Tooling | [![mise](https://img.shields.io/badge/mise-toml-1E1E2E)](./mise.toml) · Taskfile | Pinned Go / Node / Rust / protoc versions; single `./oentike` launcher |
| CI | [![GitHub Actions](https://img.shields.io/badge/CI-GitHub_Actions-2088FF?logo=github-actions&logoColor=white)](./.github/workflows/ci.yml) | Go tests + gofumpt, Astro build + atlas validation |

### Why this stack?

- **gRPC over REST** — the conditions contract carries typed factors, algorithm versions, and SHA-256 input hashes. Protobuf makes this strict without hand-rolled validation. HTTP is only for health probes.
- **PostGIS over plain Postgres** — cells are real polygons in EPSG:2180 (Poland-specific CRS). Spatial indexing drives both the map and nearest-cell lookups; this is not metadata that could live in a flat table.
- **Tauri over Electron** — the macOS host needs `objc2-core-location` for GPS in expedition mode. Tauri's Rust core lets us call Apple frameworks directly, use ed25519 for atlas pack signatures, and ship a ~15 MB binary instead of a bundled Chromium.
- **Astro over SPA** — atlas pages are static content; shipping a React runtime to display a mushroom card is waste. Astro islands (GSAP, MapLibre) hydrate only where needed.
- **mise over global installs** — Go 1.27, Node 24.12, Rust 1.97, protoc 35.1 are pinned in one file. No "works on my machine."

---

## Data Integrity

Oentike treats scoring honesty as a first-class principle — not a footnote.

- **Provenance** — every weather sample records its `ingest_run_id`, the Open-Meteo request URL, and a SHA-256 of the response. Scores link back to the exact data that produced them.
- **Explicit unknowns** — missing weather → status `unavailable`, score `null`. The system never interpolates, guesses, or borrows from a neighboring cell. Empty days in the season chart stay empty.
- **Algorithm versioning** — every score is stamped with its `algorithm_version` (e.g. `oentike-conditions/0.1.0-boletus`). When models change, old scores remain comparable and auditable.

---

## Quick Start

### Prerequisites

| Tool | Version | Install |
|---|---|---|
| [mise](https://mise.jdx.dev) | latest | `curl https://mise.jdx.dev/install.sh \| sh` |
| [Docker](https://docs.docker.com/get-docker/) + Compose | recent | Desktop or package manager |
| Xcode CLI Tools (macOS) | — | `xcode-select --install` |

### Local Development

```bash
# 1. Clone and pin tool versions
git clone https://github.com/Niutaq/Oentike && cd Oentike
mise trust && mise install

# 2. Verify environment
./oentike doctor

# 3. Install UI dependencies
./oentike setup

# 4. Pull BDL forest districts into PostGIS (~429 nadleśnictwa)
./oentike sync-bdl

# 5. Start everything (PostGIS → API → Astro → Oentike.app)
./oentike dev
```

First run compiles Go + Rust; subsequent starts take seconds.
Use `FORCE_BUILD=1 ./oentike dev` after native code changes.

### Services

| Service | Address |
|---|---|
| UI (Astro dev) | `http://127.0.0.1:4321` |
| HTTP health / ready | `http://127.0.0.1:8081/healthz`, `/readyz` |
| gRPC Conditions | `:8082` |
| PostGIS | `127.0.0.1:54321` · database `oentike` |

### Available Commands

| Command | Description |
|---|---|
| `./oentike dev` | PostGIS + API + Astro + desktop app |
| `./oentike field` | Expedition mode (GPS + hot reload; no API) |
| `./oentike field-hot` | Naked `tauri dev` — fast Rust iteration, no Core Location on macOS |
| `./oentike setup` | Install locked web dependencies |
| `./oentike doctor` | Verify tools and Docker (no installs) |
| `./oentike check` | Go tests + atlas validation + plist contract + production build |
| `./oentike atlas` | Validate species cards and SVG assets |
| `./oentike bench` | Spatial lookup and scoring benchmarks |
| `./oentike sync-bdl` | Pull BDL nadleśnictwa into PostGIS via WFS |
| `./oentike ingest` | Refresh Open-Meteo weather for materialized cells (`CELL=id` for one) |
| `./oentike stop` | Stop PostGIS |

### Linux

Install Tauri system dependencies once before the first desktop build:

```bash
sudo apt update && sudo apt install -y libwebkit2gtk-4.1-dev build-essential \
  curl wget file libxdo-dev libssl-dev libayatana-appindicator3-dev librsvg2-dev
```

---

## Known Limitations

- **Species models are provisional** — weather envelopes only (precipitation, soil temperature, soil moisture). No mycelium growth modeling, no forest-type weighting. Confidence is always `low`. This is not forage advice.
- **Static `.app` build has empty conditions** — the API proxy only works while Astro dev is running. A packaged app with static `dist/` cannot reach the gRPC backend. Daily `./oentike dev` works; standalone `.app` does not yet.
- **Open-Meteo rate limits** — bulk ingest of all ~429 cells can hit API limits. Use `CELL=id` for targeted refresh.
- **macOS-only GPS** — Core Location requires the trusted `.app` host built by `dev`/`field`. On Linux, expedition mode has no native GPS; location depends on browser geolocation in `tauri dev`.
- **Season chart = single cell** — shows last 9 Warsaw civil days for one selected cell. No national choropleth, no cross-cell aggregation.

---

## Roadmap

- [x] Explainable per-species scoring (13 pilot taxa)
- [x] Searchable BDL nadleśnictwa (~429 forest districts)
- [x] Season trend chart (9-day history per cell)
- [x] Offline expedition journal (IndexedDB + GeoJSON)
- [x] Versioned species atlas with anatomical SVG plates
- [x] Core Location integration (macOS `.app` host)
- [x] CI pipeline (Go tests + gofumpt + Astro build + atlas validation)
- [x] i18n support (PL / EN)
- [ ] Signed atlas packs (ed25519 — Rust side ready, signing flow pending)
- [ ] PMTiles offline map cache
- [ ] NATS JetStream event streaming (score invalidation, weather sync)
- [ ] SPIFFE/SPIRE workload identity + mTLS
- [ ] Sentinel/STAC satellite habitat layer
- [ ] National season choropleth (blurred, no exact pins)
- [ ] Gemma local RAG over atlas and score explanations

---

## Security & Contributing

| | |
|---|---|
| Security | [SECURITY.md](./SECURITY.md) |
| Contributing | [CONTRIBUTING.md](./CONTRIBUTING.md) |

Atlas cards are original content. Do not copy descriptions or photographs from external atlases. Artwork uses conventional botanical illustration style and is not an identification key.

---

## License

See [LICENSE](./LICENSE) for details.

---

## Screenshots

<div align="center">

<img src="./Oentike_1.png" alt="Oentike — mushroom conditions score for Nadleśnictwo Rozwadów" width="720" />

*Conditions dashboard: explainable score with weather factors, algorithm version, and confidence level.*

<br />

<img src="./Oentike_2.png" alt="Oentike — pilot cell map and 9-day season trend" width="720" />

*Season trend chart and spatial cell map for a selected nadleśnictwo.*

</div>
