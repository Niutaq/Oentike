package conditions

//go:generate protoc -I ../../../oentike-proto --go_out=../.. --go_opt=module=oentike-api --go-grpc_out=../.. --go-grpc_opt=module=oentike-api conditions.proto
