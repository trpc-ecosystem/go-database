package goes

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/transport"
)

func TestClientTransport_RoundTrip_requestWrapper_should_exists(t *testing.T) {
	ctx := trpc.CloneContext(trpc.BackgroundContext())

	msgCtx, msg := codec.WithCloneMessage(ctx)
	rspWrapper := &responseWrapper{}
	msg.WithClientRspHead(rspWrapper)

	st := NewClientTransport()
	optNetwork := transport.WithDialNetwork("tcp")
	optPool := transport.WithDialPool(nil)
	_, err := st.RoundTrip(msgCtx, []byte("hello"), optNetwork, optPool)
	if err == nil {
		t.Fatalf("expect error, got nil")
	}
	te, ok := err.(*errs.Error)
	if !ok {
		t.Fatalf("expect *errs.Error, got %T", err)
	}

	assert.Equal(t, int(errs.RetClientEncodeFail), int(te.Code))
	assert.Equal(t, "goes get requestWrapper failed", te.Msg)
}

func TestClientTransport_RoundTrip_requestWrapper_should_not_nil(t *testing.T) {
	ctx := trpc.CloneContext(trpc.BackgroundContext())

	msgCtx, msg := codec.WithCloneMessage(ctx)
	rspWrapper := &responseWrapper{}
	reqWrapper := (*requestWrapper)(nil)
	msg.WithClientRspHead(rspWrapper)
	msg.WithClientReqHead(reqWrapper)

	st := NewClientTransport()
	optNetwork := transport.WithDialNetwork("tcp")
	optPool := transport.WithDialPool(nil)
	_, err := st.RoundTrip(msgCtx, []byte("hello"), optNetwork, optPool)
	if err == nil {
		t.Fatalf("expect error, got nil")
	}
	te, ok := err.(*errs.Error)
	if !ok {
		t.Fatalf("expect *errs.Error, got %T", err)
	}

	assert.Equal(t, int(errs.RetClientEncodeFail), int(te.Code))
	assert.Equal(t, "goes request empty", te.Msg)
}

func TestClientTransport_RoundTrip_requestWrapper_bad_request(t *testing.T) {
	ctx := trpc.CloneContext(trpc.BackgroundContext())

	msgCtx, msg := codec.WithCloneMessage(ctx)
	rspWrapper := &responseWrapper{}
	reqWrapper := &requestWrapper{request: &http.Request{}}
	msg.WithClientRspHead(rspWrapper)
	msg.WithClientReqHead(reqWrapper)

	st := NewClientTransport()
	optNetwork := transport.WithDialNetwork("tcp")
	optPool := transport.WithDialPool(nil)
	_, err := st.RoundTrip(msgCtx, []byte("hello"), optNetwork, optPool)
	if err == nil {
		t.Fatalf("expect error, got nil")
	}
	assert.Equal(t, "http: nil Request.URL", err.Error())
}

func TestClientTransport_RoundTrip_responseWrapper_should_exists(t *testing.T) {
	ctx := trpc.CloneContext(trpc.BackgroundContext())

	msgCtx, msg := codec.WithCloneMessage(ctx)
	reqWrapper := &requestWrapper{request: &http.Request{}}
	msg.WithClientReqHead(reqWrapper)

	st := NewClientTransport()
	optNetwork := transport.WithDialNetwork("tcp")
	optPool := transport.WithDialPool(nil)
	_, err := st.RoundTrip(msgCtx, []byte("hello"), optNetwork, optPool)
	if err == nil {
		t.Fatalf("expect error, got nil")
	}
	te, ok := err.(*errs.Error)
	if !ok {
		t.Fatalf("expect *errs.Error, got %T", err)
	}

	assert.Equal(t, int(errs.RetClientEncodeFail), int(te.Code))
	assert.Equal(t, "goes get responseWrapper failed", te.Msg)
}
