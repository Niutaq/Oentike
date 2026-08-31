#!/bin/bash
set -e

GREEN='\033[0;32m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}=== CyberSec Zero-Trust Architecture Demo ===${NC}"
echo "Booting core infrastructure: SPIRE, NATS, Core Gateway & WASM Proxy..."
# Clean up any stale zombie processes from previous go run . executions
pkill -f "oentike-control-plane" || true
pkill -f "test-client" || true

task nats > /dev/null 2>&1
task spire > /dev/null 2>&1
task wasm > /dev/null 2>&1

echo -e "${GREEN}[OK] Network infrastructure active.${NC}"

echo -e "\nStarting Go Control Plane (gRPC listener on :50051)..."
cd oentike-control-plane
go run . > /tmp/oentike-cp.log 2>&1 &
CP_PID=$!
cd ..
sleep 6

echo -e "Starting Envoy Proxy (L7 WASM & mTLS Gateway)..."
task envoy > /tmp/envoy.log 2>&1 &
ENVOY_PID=$!
sleep 8

echo -e "\n${CYAN}[TEST CLIENT]${NC} Establishing mTLS authentication and gRPC telemetry stream..."
cd test-client
go mod tidy > /dev/null 2>&1
go run . > /tmp/oentike-client.log 2>&1 &
CLIENT_PID=$!
cd ..
sleep 4

echo -e "${GREEN}[STREAM ESTABLISHED] Client Logs:${NC}"
tail -n 3 /tmp/oentike-client.log

echo -e "\n${RED}>>> INITIATING KILL SWITCH (ACTIVE CONTEXT CANCELLATION) <<<${NC}"
echo "SOC Analyst triggers QUARANTINE command"
# Using native Go script for 100% reliable NATS payload delivery
cd trigger
go run test_nats_pub.go
cd ..

sleep 2

echo -e "\n${CYAN}[TEST CLIENT] Reaction to TCP RST_STREAM frame:${NC}"
tail -n 6 /tmp/oentike-client.log | grep -A 2 "FATAL" || echo -e "${RED}[x] Explosion not caught (check logs).${NC}"

echo -e "\nCleaning up background processes..."
kill $CLIENT_PID || true
kill $CP_PID || true
docker rm -f oentike-envoy > /dev/null 2>&1
echo "Demo complete."
