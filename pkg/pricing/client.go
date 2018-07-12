package pricing

type PricingClient interface {
	ETHtoUSD() (float64, error)
	BTCtoETH() (float64, error)
}
