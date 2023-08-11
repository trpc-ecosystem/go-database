package kafka

// ClientConversation implements the client-side of an authentication
// conversation with a server.
// go:generate mockgen -destination=./mockkafka/scram_mock.go -package=mockkafka . ClientConversation
type ClientConversation interface {
	Step(challenge string) (response string, err error)
	Done() bool
}
