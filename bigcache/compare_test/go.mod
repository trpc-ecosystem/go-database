module test

go 1.18

replace	trpc.group/trpc-go/trpc-database/bigcache => ../
replace trpc.group/trpc-go/trpc-database/localcache => ../../localcache

require (
	trpc.group/trpc-go/trpc-database/bigcache v0.0.1
	trpc.group/trpc-go/trpc-database/localcache v0.0.1
	github.com/allegro/bigcache/v3 v3.0.0
)
