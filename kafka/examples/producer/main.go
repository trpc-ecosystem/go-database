// Package main is the main package.
// Used to test client production messages
package main

import (
	"context"
	"fmt"
	"time"

	"trpc.group/trpc-go/trpc-database/kafka"
	"trpc.group/trpc-go/trpc-go"
)

func main() {
	trpc.NewServer()
	key := "key"
	value := "value"
	proxy := kafka.NewClientProxy("trpc.kafka.producer.service")
	for i := 0; i < 3; i++ {
		err := proxy.Produce(context.Background(), []byte(key), []byte(value))
		if err != nil {
			panic(err)
		}
		fmt.Println(i, key, value)
		time.Sleep(time.Second)
	}
}
