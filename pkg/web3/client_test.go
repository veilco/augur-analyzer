package web3_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stateshape/predictions.global/server/pkg/env"
	"github.com/stateshape/predictions.global/server/pkg/web3"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
)

func TestEthereumConnection(t *testing.T) {
	ws, http := os.Getenv(env.EthereumHostWS), os.Getenv(env.EthereumHostHTTP)
	if ws == "" || http == "" {
		t.Skip(fmt.Sprintf("Skipping because `%s` and `%s` env vars are not set.", env.EthereumHostWS, env.EthereumHostHTTP))
	}

	client, err := web3.NewClient(web3.EthereumHosts{
		WS:   ws,
		HTTP: http,
	})
	assert.Nil(t, err)
	assert.NotNil(t, client)

	blockHeadersStream := make(chan *types.Header)
	sub, err := client.SubscribeNewHead(context.TODO(), blockHeadersStream)
	assert.Nil(t, err)
	select {
	case err := <-sub.Err():
		assert.Fail(t, "Connection subscribing to Ethereum block updates failed: %+v", err)
	default:
	}
	select {
	case <-blockHeadersStream:
	case <-time.After(time.Minute):
		t.Fatalf("Receiving a new block took longer than one minute")
	}
}
