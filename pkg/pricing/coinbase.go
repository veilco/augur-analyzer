package pricing

import (
	"github.com/fabioberger/coinbase-go"
)

type CoinbaseClient struct {
	client coinbase.Client
}

func NewCoinbasePricingClient(key, secret string) PricingClient {
	cb := coinbase.ApiKeyClient(key, secret)
	return &CoinbaseClient{cb}
}

func (cc *CoinbaseClient) ETHtoUSD() (float64, error) {
	return cc.client.GetExchangeRate("eth", "usd")
}

func (cc *CoinbaseClient) BTCtoETH() (float64, error) {
	return cc.client.GetExchangeRate("btc", "eth")
}
