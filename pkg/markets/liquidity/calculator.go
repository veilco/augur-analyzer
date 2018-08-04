package liquidity

import (
	"math"

	"github.com/stateshape/augur-analyzer/pkg/currency"
)

type calculator struct{}

func NewCalculator() Calculator {
	return &calculator{}
}

// Assumption: 1.0 complete sets of shares is purchased from system for 1.0 currency units
func (c *calculator) GetLiquidityRetentionRatio(sharesPerCompleteSet float64, allowance currency.Ether, books []OutcomeOrderBook) float64 {
	// Allowance needs to be in the same denomination that the orders are priced in

	// Due to rounding, it might be that case that we spend up to `0.5 * sharesPerCompleteSet` more money
	// than the `allowance` parameter says.
	completeSets := math.Round(allowance.Float64() / sharesPerCompleteSet)

	// Keep track of money made from selling complete sets
	proceeds := 0.0

	// Handles yesNo and scalar markets
	if len(books) < 2 {
		// Sell back as many of outcome[1] as possible
		proceeds += books[0].TakeBids(completeSets, TakeOptions{})
		// Sell back as many of outcome[0] as possible
		proceeds += books[0].TakeAsks(completeSets, TakeOptions{})

		return proceeds / allowance.Float64()
	}

	// Handle categorical markets
	for completeSets > 0 {
		// Since there are len(outcomes) + 1 different ways to sell the shares back into the market
		// try them each to observe their profitability.

		// Option 1: Sell all shares individually using the bids for each respective outcome.
		// Option 2: Sell one share individually using the bids in its respective orderbook, and
		//           sell the rest of the shares using the asks in the order book.

		// estimatedProceeds[i] is the proceeds from selling complete sets into the outcomes[i] order book.
		// estiamtedProceeds[len(outcomes)] is the proceeds from selling each share individually into their respective order books
		estimatedProceeds := make([]float64, len(books)+1)

		dryRun := TakeOptions{DryRun: true}
		for i := 0; i < len(books); i++ {
			estimatedProceeds[len(books)] += books[i].TakeBids(sharesPerCompleteSet, dryRun)

			estimatedProceeds[i] += books[i].TakeBids(sharesPerCompleteSet, dryRun)
			estimatedProceeds[i] += books[i].TakeAsks(sharesPerCompleteSet, dryRun)
		}

		// Determine strategy which yields the most proceeds
		maxProceeds := 0.0
		maxProceedsIndex := 0
		for i := 0; i < len(estimatedProceeds); i++ {
			if estimatedProceeds[i] > maxProceeds {
				maxProceeds = estimatedProceeds[i]
				maxProceedsIndex = i
			}
		}

		// Unable to sell into market order books
		if maxProceeds == 0 {
			break
		}

		// Execute most profitable strategy
		proceedsFromShares := 0.0

		if maxProceedsIndex == len(books) {
			for i := 0; i < len(books); i++ {
				proceedsFromShares += books[i].TakeBids(sharesPerCompleteSet, TakeOptions{})
			}
		} else {
			proceedsFromShares += books[maxProceedsIndex].TakeBids(sharesPerCompleteSet, TakeOptions{})
			proceedsFromShares += books[maxProceedsIndex].TakeAsks(sharesPerCompleteSet, TakeOptions{})
		}

		proceeds += proceedsFromShares
		completeSets -= sharesPerCompleteSet
	}

	return proceeds / allowance.Float64()
}
