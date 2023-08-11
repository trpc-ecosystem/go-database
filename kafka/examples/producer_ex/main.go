// Package main is the main package.
// Used to test client production messages
package main

import (
	"context"
	"fmt"
	"time"

	"trpc.group/trpc-go/trpc-database/kafka"
	"trpc.group/trpc-go/trpc-go/client"
)

func main() {
	proxy := kafka.NewClientProxy("trpc.kafka.server.service",
		client.WithTarget("kafka://127.0.0.1:9092?clientid=test_producer&partitioner=hash"))
	for i := 0; i < 3; i++ {
		// Synchronous producing
		partition, offset, err := proxy.SendMessage(context.Background(), "test_topic", []byte("hello"),
			[]byte("sync"))
		fmt.Printf("partition=%v\toffset=%v\terr=%v\n", partition, offset, err)
		time.Sleep(time.Second)
	}
	for i := 0; i < 3; i++ {
		//asynchronous producing
		err := proxy.AsyncSendMessage(context.Background(), "test_topic", []byte("hello"), []byte("async"))
		fmt.Printf("err=%v\n", err)
		time.Sleep(time.Second)
	}
}
