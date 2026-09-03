<div align="center">

<img src="./oentike-web/public/oentike-logo.svg" alt="Oentike logo" width="112" />

# Oentike

Local mushroom-conditions helper for Polish forests: explainable scores, a coarse seasonal map, offline atlas later.

[![Go](https://img.shields.io/badge/API-Go_gRPC-00ADD8?style=for-the-badge&logo=go)](./oentike-api)
[![PostGIS](https://img.shields.io/badge/Geo-PostGIS-336791?style=for-the-badge)](./oentike-api/migrations)
[![Tauri](https://img.shields.io/badge/Desktop-Tauri_2-FFC131?style=for-the-badge&logo=tauri)](./oentike-web)
[![SPIFFE](https://img.shields.io/badge/Identity_lab-SPIFFE%2FSPIRE-red?style=for-the-badge)](./spire)

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

Agent rules and the intended stack (H3, signed packs, SPIFFE on the product path, atlas, Gemma last): [`AI_AGENT_PROMPT.md`](./AI_AGENT_PROMPT.md).

---

## Layout

| Path | Role |
|---|---|
| `oentike-api/` | Product API: migrations, PostGIS, gRPC conditions |
| `oentike-proto/` | `conditions.proto` (product) and `fingate.proto` (identity lab) |
| `oentike-web/` | Astro UI + Tauri desktop |
| `oentike-control-plane/`, `spire/`, `envoy.yaml`, `oentike-wasm-filter/`, `test-client/`, `trigger/` | **Identity lab** - SPIFFE/mTLS/Envoy to reuse on ingest/sync later, not the default `task dev` path |
| `docker-compose.yml` | PostGIS (`profile: oentike`) plus lab services |

`task start-all` still boots the identity lab (no Python LLM, no Ollama). `./demo_kill_switch.sh` is the old quarantine demo.

---

## Tooling

Pinned in [`mise.toml`](./mise.toml): Go, Rust, protoc. Workflows in [`Taskfile.yml`](./Taskfile.yml).

```bash
mise install
mise trust
task --list
```

| Task | What |
|---|---|
| `dev` | Proto + migrate + API + Tauri |
| `oentike:api` | Health `:8081` + gRPC `:8082` |
| `oentike:ingest` | Open-Meteo → `ingest_runs` + `weather_samples` |
| `oentike:test` | Go tests |
| `oentike:ui` | Desktop only (`FRESH=1` clears Tauri cache) |
| `start-all` | Identity lab + Tauri |
