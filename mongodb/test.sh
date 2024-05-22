#!/bin/sh
go clean --modcache
# Only run unit tests of type *Unit*, filtering some test cases with network calls.
go test ./... --covermode=count -coverprofile=cover.out -v  -gcflags=all=-l -run Unit