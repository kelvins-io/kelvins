package middleware

import (
	"context"
	"testing"
)

func TestRpcCredentials_GetRequestMetadata(t *testing.T) {
	x := rpcAuthPerCredentials{
		sharedSecret: "c9VW6ForlmzdeDkZE2i8",
	}
	m, err := x.GetRequestMetadata(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Log(m)
}
