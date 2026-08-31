module test-client

go 1.27.0

replace oentike-control-plane => ../oentike-control-plane

require (
	github.com/nats-io/nats.go v1.53.1
	github.com/spiffe/go-spiffe/v2 v2.8.1
	google.golang.org/grpc v1.83.2
	oentike-control-plane v0.0.0-00010101000000-000000000000
)

require (
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/go-jose/go-jose/v4 v4.1.4 // indirect
	github.com/klauspost/compress v1.18.5 // indirect
	github.com/nats-io/nkeys v0.4.15 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	golang.org/x/crypto v0.55.0 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260825221802-da73d73af1c5 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)
