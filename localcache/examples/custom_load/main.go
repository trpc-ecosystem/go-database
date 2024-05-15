package main

import (
	"context"
	"fmt"

	"trpc.group/trpc-go/trpc-database/localcache"
)

func main() {
	mLoadFunc := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		return map[string]interface{}{
			"foo": "bar",
			"tom": "cat",
		}, nil
	}

	lc := localcache.New(localcache.WithMLoad(mLoadFunc), localcache.WithExpiration(5))

	// err is the error message returned directly by the mLoadFunc function.
	val, err := lc.MGetWithLoad(context.TODO(), []string{"foo", "tom"})
	fmt.Println(val, err)

	// Or you can pass in the load function at get time
	val, err = lc.MGetWithCustomLoad(context.TODO(), []string{"foo"}, mLoadFunc, 10)
	fmt.Println(val, err)
}
