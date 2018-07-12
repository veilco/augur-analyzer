package web3

import (
	"github.com/ethereum/go-ethereum/ethclient"
)

type EthereumHosts struct {
	WS   string
	HTTP string
}

// NewClient creates a new Web3 client
func NewClient(hosts EthereumHosts) (*ethclient.Client, error) {
	return ethclient.Dial(hosts.HTTP)
}
