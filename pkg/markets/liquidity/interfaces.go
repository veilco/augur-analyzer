package liquidity

import (
	"github.com/stateshape/augur-analyzer/pkg/currency"
)

type OutcomeOrderBook interface {
	TakeBids(maxSharesToTake float64, options TakeOptions) (proceeds float64)
	TakeAsks(maxSharesToTake float64, options TakeOptions) (proceeds float64)
}

type Calculator interface {
	GetLiquidityRetentionRatio(sharesPerCompleteSet float64, allowance currency.Ether, outcomes []OutcomeOrderBook) (retentionRatio float64)
}

type TakeOptions struct {
	DryRun bool
}
