package goes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/trpc-go/errs"
)

// TestUnitImpSetup test Setup
func TestUnitImpSetup(t *testing.T) {
	p := Plugin{}
	err := p.Setup(pluginName, &Decoder{})
	assert.Nil(t, err)
}

// Decoder is a mock config decoder
type Decoder struct{}

// Decode config decoder
func (c *Decoder) Decode(cfg interface{}) error {
	cfgIns := cfg.(*Config)
	cfgIns.ClientsOptions = []*ClientOption{{Name: "g1"}, {Name: "g2"}}
	return nil
}

// TestUnitImpFailDecode test FailDecode
func TestUnitImpFailDecode(t *testing.T) {
	p := Plugin{}
	err := p.Setup(pluginName, &FailDecoder{})
	assert.NotNil(t, err)
}

// FailDecoder is a mock config decoder with fail decode
type FailDecoder struct{}

// Decode config decoder
func (c *FailDecoder) Decode(cfg interface{}) error {
	return errs.New(-1, "failed to decode")
}
