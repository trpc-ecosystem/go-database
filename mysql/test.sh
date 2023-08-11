#!/bin/sh
go clean --modcache
# Run only *Unit* type unit tests and filter some test cases with network calls.
go test ./... --covermode=count -coverprofile=cover.out -v  -gcflags=all=-l -run Unit