// Package main is the main package.
// Used to test server consumption of multiple messages
package main

import (
	"context"
	"fmt"

	"github.com/Shopify/sarama"
	"trpc.group/trpc-go/trpc-database/kafka"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/log"
)

func main() {
	s := trpc.NewServer()
	kafka.RegisterBatchHandlerService(s, Handle)

	go Produce()

	err := s.Serve()
	if err != nil {
		panic(err)
	}
}

// Produce producing
func Produce() {
	ctx := context.TODO()
	for i := 0; i < 17; i++ {
		key := fmt.Sprintf("key_%d", i)
		value := fmt.Sprintf("value_%d", i)
		proxy := kafka.NewClientProxy("trpc.kafka.producer.service")
		p, offset, err := proxy.SendMessage(ctx, "test", []byte(key), []byte(value))
		if err != nil {
			log.ErrorContextf(ctx, "send msg failed, %s", err)
			return
		}
		log.Infof("[produce][partition]%v\t[offset]%v\t[key]%v\t[value]%v\n",
			p, offset, key, value)
	}
}

// Handle handle function
func Handle(ctx context.Context, msgArray []*sarama.ConsumerMessage) error {
	log.Infof("len(msgArray) = %d", len(msgArray))
	for _, v := range msgArray {
		log.Infof("[consume][topic]%v\t[partition]%v\t[offset]%v\t[key]%v\t[value]%v\n",
			v.Topic, v.Partition, v.Offset, string(v.Key), string(v.Value))
	}
	return nil
}
