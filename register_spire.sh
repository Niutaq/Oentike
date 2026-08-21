#!/bin/bash
set -e

echo "Registering Node and Agents"

# 1. Wipe old state and start the server
sudo docker compose rm -s -v -f spire-server spire-agent
sudo env SPIRE_JOIN_TOKEN="" docker compose up -d spire-server
sleep 3

# Step 1: Generate Join Token for the Agent
echo "Generating token for SPIRE Agent..."
TOKEN=$(sudo docker exec zt-spire-server bin/spire-server token generate -spiffeID spiffe://example.org/my-node | awk '{print $2}')
echo "Generated Token: $TOKEN"

# Prepare directory for sockets
mkdir -p ./spire/sockets
chmod 777 ./spire/sockets

# Symlink for host processes that expect it at /tmp
sudo rm -rf /tmp/spire-sockets
sudo ln -s $(pwd)/spire/sockets /tmp/spire-sockets

# Step 2: Start the Agent with the Token
echo "Registering zt-spire-agent using token..."
sudo env SPIRE_JOIN_TOKEN="$TOKEN" docker compose up -d spire-agent
sleep 3 # Wait for the Agent to start and create the socket

echo "Granting socket permissions for local processes..."
sudo chmod 777 /tmp/spire-sockets/workload_api.sock || true

# Get the current user UID under which Go and Rust processes will run (on the host)
USER_UID=$(id -u)

# Step 3: Register common Workload for local processes
echo "Registering identity for local processes (UID: $USER_UID)..."
sudo docker exec zt-spire-server bin/spire-server entry create \
    -parentID spiffe://example.org/my-node \
    -spiffeID spiffe://example.org/fingate \
    -selector unix:uid:$USER_UID \
    -dns localhost || true

echo "Registering identity for Envoy container (UID: 101)..."
sudo docker exec zt-spire-server bin/spire-server entry create \
    -parentID spiffe://example.org/my-node \
    -spiffeID spiffe://example.org/fingate \
    -selector unix:uid:101 \
    -dns localhost || true

echo "Registering identity for root containers (UID: 0)..."
sudo docker exec zt-spire-server bin/spire-server entry create \
    -parentID spiffe://example.org/my-node \
    -spiffeID spiffe://example.org/fingate \
    -selector unix:uid:0 \
    -dns localhost || true

echo "== Successfully registered =="
echo "Available identities:"
sudo docker exec zt-spire-server bin/spire-server entry show || true

echo "== SPIRE Agent Logs =="
sudo docker logs zt-spire-agent || true
