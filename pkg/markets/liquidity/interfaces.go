package liquidity

import (
	"github.com/stateshape/augur-analyzer/pkg/currency"
)

type OutcomeOrderBook interface {
	DeepClone() OutcomeOrderBook
	CloseLongFillOnly(shares float64, market MarketData, dryRun bool) (proceeds float64)
	CloseShortFillOnly(shares float64, market MarketData, dryRun bool) (proceeds float64)
}

type Calculator interface {
	GetLiquidityRetentionRatio(sellingIncrement float64, allowance currency.Ether, market MarketData, outcomes []OutcomeOrderBook) (retentionRatio float64)
}

type MarketData struct {
	MinPrice float64
	MaxPrice float64
}
