# Oentike (FinOps Proxy) - Agent Context & Architecture Manifesto

**To the AI Agent reading this:** This file is your primary source of truth for the Oentike project. Read it carefully before making architectural decisions. The user expects high-quality engineering, no generic "AI-slop" comments in code, and a focus on both learning niche technologies and practical business applications.

## 1. Project Overview & Business Logic
Oentike is a distributed **Zero-Trust FinOps proxy system**. Its main goal is to stream hardware telemetry from edge agents, route it through an L7 firewall (Envoy + WASM), process it centrally (Go Control Plane), and use an AI Orchestrator (Python + Ollama) to analyze metrics and reject/approve workloads based on a FinOps budget (FOCUS standard).

### The "Why" (Business & Engineering)
- **Business:** Organizations waste money on cloud computing. Oentike monitors telemetry in real-time, estimates costs, and dynamically cuts off expensive workloads using AI. We stick strictly to the **FOCUS (FinOps)** standard for metrics.
- **Engineering / Learning:** The stack is highly niche and intentionally over-engineered in certain areas for the sake of learning modern cloud-native & edge paradigms.

## 2. Technology Stack
This is a polyglot monorepo. It relies on `task` (Taskfile.yml) and `nix-shell` (or `mise`) for environment reproducibility.

- **`oentike-edge-agent` (Rust + iced + tonic)**
  - Desktop application mimicking a hardware sensor.
  - Fetches X.509 SVIDs from SPIRE.
  - Streams telemetry to the control plane via gRPC over mTLS.
- **`oentike-wasm-filter` (Rust + Proxy-WASM)**
  - Compiled to `wasm32-wasip1`.
  - Runs inside Envoy as an L7 filter (FinOps Tollbooth).
  - Inspects gRPC metadata (`x-finops-cost`, `x-finops-budget`). Drops requests (HTTP 402) if cost > budget.
- **`Envoy` & `SPIRE` (C++ / Go, Docker)**
  - SPIRE provides identity (SPIFFE IDs) to processes and containers.
  - Envoy terminates mTLS, enforcing Zero-Trust using SPIRE certificates.
- **`oentike-control-plane` (Go + SQLite)**
  - gRPC server receiving telemetry.
  - Converts telemetry to FOCUS-standard FinOps events.
  - Publishes events to NATS JetStream.
  - Contains an `Approval Worker` that listens for AI decisions from NATS and saves them to SQLite.
- **`oentike-ai-orchestrator` (Python + LangGraph + Ollama)**
  - Subscribes to NATS.
  - Uses local LLMs (e.g., `qwen3-vl:4b`) to analyze telemetry.
  - Emits routing/approval decisions (`REJECTED_BUDGET`, `ROUTED`) back to NATS.
- **`oentike-web` (Astro + Vue/React/Svelte)**
  - Frontend dashboard displaying FinOps metrics.

## 3. Future Expansion & Rules of Thumb

### What makes sense (DO THIS):
1. **Maintain the FOCUS standard:** Any new metrics or cost reporting must adhere to FinOps Open Cost and Usage Specification (FOCUS).
2. **Zero-Trust first:** Any new component must authenticate via SPIFFE/SPIRE.
3. **Reproducibility:** If you add a dependency, add it to `Taskfile.yml`, `mise.toml`, or `register_spire.sh`.
4. **NATS JetStream:** Use NATS for all asynchronous/event-driven communication between microservices.
5. **Code purity:** Write clean, idiomatic code for each language (e.g., proper error handling in Rust, channels/goroutines in Go). Keep comments strictly informative (explain *why*, not *what*). **ABSOLUTELY NO AI-SLOP COMMENTS.**

### What doesn't make sense (AVOID):
1. **Monolithic architectures:** Do not try to merge the Go control plane and Python AI into one app. They are separate for a reason.
2. **Bypassing Envoy:** Do not expose the Go gRPC server directly to the Rust agent over plaintext. Always route through Envoy mTLS.
3. **Heavy cloud dependencies:** This project is designed to run locally (edge/local cluster) using Ollama for AI. Avoid hardcoding AWS/GCP services unless specifically requested.

## 4. How to run
```bash
task start-all
```
*(Requires sudo for Docker, nix-shell, and Spire socket permissions)*
