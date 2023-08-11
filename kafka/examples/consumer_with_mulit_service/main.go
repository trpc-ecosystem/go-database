// Package main is the main package.
package main

import (
	"context"

	"github.com/Shopify/sarama"
	"trpc.group/trpc-go/trpc-database/kafka"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/log"
)

func main() {
	s := trpc.NewServer()
	// The parameter of s.Service should be the same as the name of the service in
	// the configuration file. The configuration is loaded based on this parameter.
	kafka.RegisterKafkaConsumerService(s.Service("trpc.databaseDemo.kafka.consumer1"), &Consumer{})
	kafka.RegisterKafkaConsumerService(s.Service("trpc.databaseDemo.kafka.consumer2"), &Consumer{})
	s.Serve()
}

// Consumer is the consumer
type Consumer struct{}

// Handle is handle function
func (Consumer) Handle(ctx context.Context, msg *sarama.ConsumerMessage) error {
	log.Infof("get kafka msg: %+v", msg)
	// Successful consumption is confirmed only when returning nil.
	return nil
}
