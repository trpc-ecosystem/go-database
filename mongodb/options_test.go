package mongodb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestProduceOption(t *testing.T) {
	pm := &event.PoolMonitor{}
	i := func(dsn string, opts *options.ClientOptions) {
		opts.SetPoolMonitor(pm)
	}
	transport := NewMongoTransport(WithOptionInterceptor(i))
	ct := transport.(*ClientTransport)
	opts := options.ClientOptions{}
	ct.optionInterceptor("", &opts)
	assert.Equal(t, opts.PoolMonitor, pm)
}
