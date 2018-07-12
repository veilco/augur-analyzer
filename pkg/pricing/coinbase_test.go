package pricing_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/stateshape/augur-analyzer/pkg/env"
	"github.com/stateshape/augur-analyzer/pkg/pricing"

	"github.com/stretchr/testify/assert"
)

func TestCoinbasePricingClient(t *testing.T) {
	key, secret := os.Getenv(env.CoinbaseAPIKey), os.Getenv(env.CoinbaseAPISecret)
	if key == "" || secret == "" {
		t.Skip(fmt.Sprintf("Skipping because `%s` and `%s` env vars are not set.", env.CoinbaseAPIKey, env.CoinbaseAPISecret))
	}
	client := pricing.NewCoinbasePricingClient(key, secret)
	_, err := client.ETHtoUSD()
	assert.Nil(t, err)
}
