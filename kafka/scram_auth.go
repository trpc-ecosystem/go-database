// Package kafka security verification class
package kafka

import (
	"crypto/sha256"
	"crypto/sha512"
	"hash"

	"github.com/IBM/sarama"
	"github.com/xdg-go/scram"
	"trpc.group/trpc-go/trpc-go/errs"
)

// SHA256 hash protocol
var SHA256 scram.HashGeneratorFcn = func() hash.Hash { return sha256.New() }

// SHA512 hash protocol
var SHA512 scram.HashGeneratorFcn = func() hash.Hash { return sha512.New() }

// SASLTypeSSL represents the SASL_SSL security protocol.
const SASLTypeSSL = "SASL_SSL"

// LSCRAMClient scram authentication client configuration
type LSCRAMClient struct {
	*scram.Client                 // client
	ClientConversation            // client session layer
	scram.HashGeneratorFcn        // hash value generating function
	User                   string // user
	Password               string // password
	Mechanism              string // encryption protocol type
	Protocol               string // encryption protocol
}

// Begin SCRAM authentication start interface
func (s *LSCRAMClient) Begin(userName, password, authzID string) (err error) {
	s.Client, err = s.HashGeneratorFcn.NewClient(userName, password, authzID)
	if err != nil {
		return err
	}
	s.ClientConversation = s.Client.NewConversation()
	return nil
}

// Step SCRAM authentication step interface
func (s *LSCRAMClient) Step(challenge string) (string, error) {
	return s.ClientConversation.Step(challenge)
}

// Done SCRAM authentication end interface
func (s *LSCRAMClient) Done() bool {
	return s.ClientConversation.Done()
}

// config config sarama client
func (s *LSCRAMClient) config(config *sarama.Config) error {
	if s == nil {
		return nil
	}
	// s is not nil, which means it has been initialized
	if len(s.Mechanism) == 0 {
		return errs.NewFrameError(errs.RetClientRouteErr, "kafka scram_client.config failed, Mechanism.len=0")
	}

	config.Net.SASL.Enable = true
	config.Net.SASL.User = s.User
	config.Net.SASL.Password = s.Password

	switch s.Protocol {
	case SASLTypeSSL:
		config.Net.SASL.Handshake = true
		config.Net.TLS.Enable = true
	default:
	}

	config.Net.SASL.Mechanism = sarama.SASLMechanism(s.Mechanism)
	switch s.Mechanism {
	case sarama.SASLTypeSCRAMSHA512:
		config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
			return &LSCRAMClient{HashGeneratorFcn: SHA512}
		}
	case sarama.SASLTypeSCRAMSHA256:
		config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
			return &LSCRAMClient{HashGeneratorFcn: SHA256}
		}
	case sarama.SASLTypePlaintext:
		return nil
	default:
		return errs.NewFrameError(errs.RetClientRouteErr, "kafka scram_client.config failed,x.mechanism "+
			"unknown("+s.Mechanism+")")
	}

	return nil
}

// Parse SCRAM local analysis
func (s *LSCRAMClient) Parse(vals []string) {
	switch vals[0] {
	case "user":
		s.User = vals[1]
	case "password":
		s.Password = vals[1]
	case "mechanism":
		s.Mechanism = vals[1]
	case "protocol":
		s.Protocol = vals[1]
	default:
	}
}
