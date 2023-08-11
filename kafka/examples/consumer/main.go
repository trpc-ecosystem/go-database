// Package main is the main package.
// Used to test server consumption messages
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
	// If you're using a custom address configuration,
	// configure it before starting the server.
	/*
		cfg := kafka.GetDefaultConfig()
		cfg.ClientID = "newClientID"
		kafka.RegisterAddrConfig("address", cfg)
	*/
	// default service name is trpc.kafka.consumer.service
	kafka.RegisterKafkaConsumerService(s, &Consumer{})

	s.Serve()
}

// Consumer is the consumer
type Consumer struct{}

// Handle handle function
func (Consumer) Handle(ctx context.Context, msg *sarama.ConsumerMessage) error {
	if rawContext, ok := kafka.GetRawSaramaContext(ctx); ok {
		log.Infof("InitialOffset:%d", rawContext.Claim.InitialOffset())
	}
	log.Infof("get kafka message: %+v", msg)
	// Successful consumption is confirmed only when returning nil.
	return nil
}
