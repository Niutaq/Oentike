<div align="center">

<img src="./oentike-web/public/oentike-wordmark.svg" alt="Oentike" width="192" />

Local mushroom-conditions helper for Polish forests: explainable scores, a coarse seasonal map, offline atlas later.

[![Go](https://img.shields.io/badge/API-Go_gRPC-00ADD8?style=for-the-badge&logo=go)](./oentike-api)
[![PostGIS](https://img.shields.io/badge/Geo-PostGIS-336791?style=for-the-badge)](./oentike-api/migrations)
[![Tauri](https://img.shields.io/badge/Desktop-Tauri_2-FFC131?style=for-the-badge&logo=tauri)](./oentike-web)

<p>
<img src="./Oentike_1.png" alt="Oentike: score and weather factors for Lasy Janowskie" width="720" />
</p>
<p>
<img src="./Oentike_2.png" alt="Oentike: pilot cell map and 9-day season trend" width="720" />
</p>

</div>

---

## Product

Pilot: searchable **BDL nadleśnictwa** (~429 after `task oentike:sync-bdl`), default **Nadleśnictwo Janów Lubelski** (`nadl-05-31`), species `boletus-edulis`. UI typeahead calls `SearchCells` over `forest_units`; selecting an area materializes it into `cells` and ingests Open-Meteo on demand (`EnsureCell` / `GetConditions`). Score color tracks 0-100 on the panel. `task oentike:ingest` refreshes weather for **materialized** cells only (`CELL=id` for one); while `serve` is up the background loop refreshes each due materialized cell about once an hour (skipped per cell if the last fetch is younger than 50 minutes). `GetConditions` scores with `oentike-conditions/0.1.0-boletus` when all three factors exist and returns `fetched_at` of that ingest. `GetSeason` returns the last 9 Warsaw days of **our** scores for the selected cell - empty days stay `unavailable`, never a fake Poland heatmap.

## Quick Start

If you have `mise` installed ([mise.jdx.dev](https://mise.jdx.dev)), you can set up and run the application with:

```bash
mise install
mise trust
mise exec -- task setup
mise exec -- task oentike:sync-bdl
mise exec -- task dev
```
*(If you have `mise` activated in your shell, you can simply run `task setup`, `task oentike:sync-bdl`, and `task dev`)*

> **Note for Linux users:** Tauri requires specific system dependencies to compile the desktop app. On Ubuntu/Debian-based distributions, install them before running `task dev`:
> ```bash
> sudo apt update && sudo apt install -y libwebkit2gtk-4.1-dev build-essential curl wget file libxdo-dev libssl-dev libayatana-appindicator3-dev librsvg2-dev
> ```


| What | Where |
|---|---|
| HTTP health / ready | `http://127.0.0.1:8081/healthz`, `/readyz` |
| Conditions RPC | gRPC `:8082` `GetConditions`, `GetSeason`, `SearchCells`, `EnsureCell` |
| PostGIS | `127.0.0.1:54321` database `oentike` |

```bash
grpcurl -plaintext -d '{"cell_id":"nadl-05-31"}' \
  127.0.0.1:8082 oentike.conditions.ConditionsService/GetConditions

grpcurl -plaintext -d '{"cell_id":"nadl-05-31"}' \
  127.0.0.1:8082 oentike.conditions.ConditionsService/GetSeason

grpcurl -plaintext -d '{"query":"Janów"}' \
  127.0.0.1:8082 oentike.conditions.ConditionsService/SearchCells

grpcurl -plaintext -d '{"cell_id":"nadl-05-31"}' \
  127.0.0.1:8082 oentike.conditions.ConditionsService/EnsureCell
```

Stop UI/API with `Ctrl+C`. Stop PostGIS with `task oentike:down`.

---

## Layout

| Path | Role |
|---|---|
| `oentike-api/` | PostGIS migrations, gRPC conditions, Open-Meteo ingest, BDL WFS sync |
| `oentike-proto/` | `conditions.proto` |
| `oentike-web/` | Astro UI + Tauri desktop (`/` conditions, `/atlas` cards) |
| `oentike-atlas/` | Versioned species cards (pilot pack, no scores) |
| `docker-compose.yml` | PostGIS (`profile: oentike`) |

---

## Tooling

Pinned in [`mise.toml`](./mise.toml): Go, Rust, protoc, task. `task --list` is the list.

```bash
mise install
mise trust
mise exec -- task --list
```

| Task | What |
|---|---|
| `dev` | Proto + migrate + API + Tauri |
| `oentike:up` / `oentike:down` | Start / stop PostGIS |
| `oentike:migrate` | Goose migrations |
| `oentike:proto` | Generate gRPC stubs |
| `oentike:sync-bdl` | Pull BDL `Nadleśnictwa` into `forest_units` (~429) |
| `oentike:api` | Health `:8081` + gRPC `:8082` (hourly Open-Meteo ingest in the background) |
| `oentike:ingest` | One-shot Open-Meteo for materialized cells → `ingest_runs` + `weather_samples` |
| `oentike:test` | Go tests |
| `oentike:ui` | Desktop only (`FRESH=1` clears Tauri cache) |
| `setup` | npm + rustup if missing |
| `clean` | `cargo clean` in Tauri (after moving the repo) |
