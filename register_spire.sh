#!/bin/bash
set -e

echo "Registering Node and Agents"

# 1. Wipe old state and start the server and Jaeger
docker compose rm -s -v -f spire-server spire-agent jaeger nats
env SPIRE_JOIN_TOKEN="" docker compose up -d spire-server jaeger nats
sleep 3

# Step 1: Generate Join Token for the Agent
echo "Generating token for SPIRE Agent..."
TOKEN=$(docker exec oentike-spire-server bin/spire-server token generate -spiffeID spiffe://example.org/my-node | awk '{print $2}')
echo "Generated Token: $TOKEN"

# Prepare directory for sockets
mkdir -p ./spire/sockets
chmod 777 ./spire/sockets

# Create native macOS socket directory
rm -rf /tmp/spire-sockets
mkdir -p /tmp/spire-sockets
chmod 777 /tmp/spire-sockets

# Run socat proxy in docker to bridge socket to TCP
echo "Starting socat proxy in Docker for SPIRE socket..."
docker rm -f oentike-spire-socat-proxy || true
# Stop any old container that might be squatting on ports
docker rm -f spire-socat-proxy || true

# Step 2: Start the Agent with the Token (starts the container that CREATES the socket in the volume)
echo "Registering oentike-spire-agent using token..."
env SPIRE_JOIN_TOKEN="$TOKEN" docker compose up -d spire-agent
sleep 3 # Wait for the Agent to start and create the socket

# Start proxy AFTER socket is created by the agent
docker run -d --rm --name oentike-spire-socat-proxy \
  -v spire-sockets-vol:/sockets \
  -p 8082:8082 \
  alpine/socat tcp-listen:8082,fork,reuseaddr unix-connect:/sockets/workload_api.sock

echo "Starting native socat proxy on host..."
pkill -f "socat unix-listen:/tmp/spire-sockets/workload_api.sock" || true
nohup socat unix-listen:/tmp/spire-sockets/workload_api.sock,fork,reuseaddr tcp:127.0.0.1:8082 >/dev/null 2>&1 &

echo "Granting socket permissions for local processes..."
chmod 777 /tmp/spire-sockets/workload_api.sock || true

# Get the current user UID under which Go and Rust processes will run (on the host)
USER_UID=$(id -u)

# Step 3: Register common Workload for local processes
echo "Registering identity for local processes (UID: $USER_UID)..."
docker exec oentike-spire-server bin/spire-server entry create \
    -parentID spiffe://example.org/my-node \
    -spiffeID spiffe://example.org/fingate \
    -selector unix:uid:$USER_UID \
    -dns localhost || true

echo "Registering identity for Envoy container (UID: 101)..."
docker exec oentike-spire-server bin/spire-server entry create \
    -parentID spiffe://example.org/my-node \
    -spiffeID spiffe://example.org/fingate \
    -selector unix:uid:101 \
    -dns localhost || true

echo "Registering identity for root containers (UID: 0)..."
docker exec oentike-spire-server bin/spire-server entry create \
    -parentID spiffe://example.org/my-node \
    -spiffeID spiffe://example.org/fingate \
    -selector unix:uid:0 \
    -dns localhost || true

echo "== Successfully registered =="
echo "Available identities:"
docker exec oentike-spire-server bin/spire-server entry show || true

echo "== SPIRE Agent Logs =="
docker logs oentike-spire-agent || true
