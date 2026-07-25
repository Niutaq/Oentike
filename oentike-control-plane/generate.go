package main

//go:generate protoc -I ../oentike-proto --go_out=. --go_opt=module=oentike-control-plane --go-grpc_out=. --go-grpc_opt=module=oentike-control-plane fingate.proto
