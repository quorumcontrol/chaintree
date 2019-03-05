package chaintree

//go:generate protoc -I=. -I=$GOPATH/src --go_out=plugins=grpc:. transactions.proto

// RUN `go generate` in this directory when updating the transactions.proto. Requires the protoc command to be in your path.
