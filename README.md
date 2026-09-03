<div align="center">

<img src="./oentike-web/public/oentike-wordmark.svg" alt="Oentike" width="220" />

Local mushroom-conditions helper for Polish forests: explainable scores, a coarse seasonal map, offline atlas later.

[![Go](https://img.shields.io/badge/API-Go_gRPC-00ADD8?style=for-the-badge&logo=go)](./oentike-api)
[![PostGIS](https://img.shields.io/badge/Geo-PostGIS-336791?style=for-the-badge)](./oentike-api/migrations)
[![Tauri](https://img.shields.io/badge/Desktop-Tauri_2-FFC131?style=for-the-badge&logo=tauri)](./oentike-web)

<img src="./Oentike.png" alt="Oentike — conditions view for Lasy Janowskie" width="100%" />

</div>

---

## Product

Pilot: one 10 km cell **Lasy Janowskie** (Targowisko / Studzieniec), species `boletus-edulis`. `task oentike:ingest` writes Open-Meteo hours into PostGIS. `GetConditions` can fill factor values from those samples, but still returns `status: unavailable` with **no score** until a scoring algorithm exists.

```bash
task dev
```

| What | Where |
|---|---|
| HTTP health / ready | `http://127.0.0.1:8081/healthz`, `/readyz` |
| Conditions RPC | gRPC `:8082` `oentike.conditions.ConditionsService/GetConditions` |
| PostGIS | `127.0.0.1:5432` database `oentike` |

```bash
grpcurl -plaintext -d '{"cell_id":"lasy-janowskie-01"}' \
  127.0.0.1:8082 oentike.conditions.ConditionsService/GetConditions
```

Stop UI/API with `Ctrl+C`. Stop PostGIS with `task oentike:down`.

---

## Layout

| Path | Role |
|---|---|
| `oentike-api/` | PostGIS migrations, gRPC conditions, Open-Meteo ingest |
| `oentike-proto/` | `conditions.proto` |
| `oentike-web/` | Astro UI + Tauri desktop |
| `docker-compose.yml` | PostGIS |

---

## Tooling

Pinned in [`mise.toml`](./mise.toml): Go, Rust, protoc. `task --list` is the list.

```bash
mise install
mise trust
task --list
```

| Task | What |
|---|---|
| `dev` | Proto + migrate + API + Tauri |
| `oentike:up` / `oentike:down` | Start / stop PostGIS |
| `oentike:migrate` | Goose migrations |
| `oentike:proto` | Generate gRPC stubs |
| `oentike:api` | Health `:8081` + gRPC `:8082` |
| `oentike:ingest` | Open-Meteo → `ingest_runs` + `weather_samples` |
| `oentike:test` | Go tests |
| `oentike:ui` | Desktop only (`FRESH=1` clears Tauri cache) |
| `setup` | npm + rustup if missing |
| `clean` | `cargo clean` in Tauri (after moving the repo) |
