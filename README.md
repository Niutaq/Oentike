<div align="center">
  
# Oentike
**Cost Control & Governance Platform**

![Current State (21-08-2026)](./state_21-08-2026.png)

[![Rust](https://img.shields.io/badge/Edge_Agent-orange?style=for-the-badge&logo=rust)](https://rust-lang.org)
[![Go](https://img.shields.io/badge/Control_Plane-blue?style=for-the-badge&logo=go)](https://golang.org)
[![Python](https://img.shields.io/badge/AI_Orchestrator-yellow?style=for-the-badge&logo=python)](https://python.org)
[![NATS](https://img.shields.io/badge/Event_Bus-NATS_JetStream-green?style=for-the-badge&logo=nats)](https://nats.io)
[![SPIFFE](https://img.shields.io/badge/Security-SPIFFE%20%2F%20mTLS-red?style=for-the-badge)](https://spiffe.io)

*Modern IT encounters two critical challenges: uncontrollable cloud waste and unpredictable decisions. Oentike solves both by establishing a cryptographically secure, multi-agent gateway.*

---
</div>

## The Vision

In an era of decentralized infrastructure, granting raw API access for cloud provisioning is a massive risk. ZT-FinGate acts as an intelligent, intercepting gateway that evaluates financial requests (such as scaling an EC2 cluster or provisioning managed databases) before they reach the cloud provider. 

Instead of static rules, it uses a Multi-Agent LLM System to analyze requests based on dynamic budgets and business context. To ensure absolute security, it implements a strict Zero-Trust network architecture (SPIFFE/mTLS), guaranteeing that only cryptographically verified applications can submit requests.

## Core Architecture

### 1. Zero-Trust Networking (SPIFFE/SPIRE & mTLS)
No application is trusted by default, regardless of its network location. 
* **Workload Identity:** Every service (from the Rust Edge Agent to the Go Control Plane) is issued a cryptographic identity via a local SPIRE server.
* **Mutual TLS (mTLS):** All gRPC communication is secured. Without a valid SPIFFE ID, the connection is instantly dropped at the transport layer.
* **How to run:** Before starting the services, you must ensure the SPIRE server is running and identities are registered (e.g., using the `register_spire.sh` script).

### 2. Multi-Agent AI Governance & ChatOps (Qwen 3.5)
AI models should not govern financial resources without strict oversight. We utilize **Qwen 3.5 (1.4b)** running locally via **Ollama** for maximum privacy and performance. The system uses LangGraph to construct a conversational dual-agent workflow:
* **Conversational FinOps Interface:** Instead of static forms, users interact with the system via a natural chat interface on both Desktop (Rust) and Web (Astro) platforms.
* **Analyst Agent:** Extracts context, cost, and risk parameters from the raw chat request using the FOCUS standard.
* **Auditor Agent:** Critiques the Analyst's decision, checks for hallucinations and policy violations, and returns a beautiful JSON response for real-time dashboard visualization.

### 3. FinOps Standard (FOCUS)
All financial events are normalized to the FOCUS (FinOps Open Cost & Usage Specification) standard. Costs are tracked, categorized, and streamed in real-time, providing total visibility into the operational burn rate.

### 4. Event-Driven Backbone (NATS JetStream)
Microservices communicate asynchronously via NATS JetStream. This provides an immutable, replayable audit log of every financial decision and infrastructure change, enabling robust tracing and real-time dashboarding via WebSockets.

## Technology Stack
* **Edge Data Plane:** Rust, Iced (GUI Chat), Tonic (gRPC)
* **Control Plane:** Go, GORM, SQLite, NATS JetStream
* **AI Orchestration:** Python, LangGraph, LangChain, Ollama (Qwen 3.5 1.4b)
* **Observability & UI:** Astro, Vanilla CSS, Native WebSockets

## System Workflow

1. A developer uses the Rust Edge Agent to chat with the system (e.g., "We need 5 new EC2 instances for the holiday season").
2. The request is cryptographically signed and sent via mTLS to the Go Control Plane.
3. The Gateway verifies the SPIFFE ID and publishes the text event to the NATS cluster.
4. The Python AI Orchestrator consumes the event. The Qwen 3.5 Analyst Agent extracts data and evaluates the business justification.
5. The Qwen 3.5 Auditor Agent verifies the decision for compliance. If safe, it approves the request.
6. The state change is streamed instantly via WebSockets to the Astro Dashboard, rendering rich FinOps visualizations.

---
*Developed as a comprehensive proof-of-concept for Cloud Architecture, FinOps, and DevSecOps.*
