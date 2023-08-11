// Package proto goredis is the library dependent proto contract file,
// and the pb.go file is generated according to proto.
package proto

// Generate pb.go file according to proto.
//go:generate trpc create --rpconly --gotag --mock=false --alias -p=goredis.proto --protodir=. -o=. --api-version 2
