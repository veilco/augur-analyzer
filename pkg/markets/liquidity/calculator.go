package liquidity

import (
	"math"

	"github.com/stateshape/augur-analyzer/pkg/currency"
)

type calculator struct{}

func NewCalculator() Calculator {
	return &calculator{}
}

// Allowance needs to be in the same denomination that the orders are priced in
func (c *calculator) GetLiquidityRetentionRatio(sellingIncrement float64, allowance currency.Ether, market MarketData, books []OutcomeOrderBook) float64 {
	// No rounding
	completeSets := allowance.Float64() / (market.MaxPrice - market.MinPrice)

	// Keep track of money made from selling complete sets
	totalProceeds := 0.0

	// Handles yesNo and scalar markets
	if len(books) < 2 {
		totalProceeds += books[0].CloseLongFillOnly(completeSets, market, false)
		totalProceeds += books[0].CloseShortFillOnly(completeSets, market, false)
		return totalProceeds / allowance.Float64()
	}

	// Handle categorical markets
	for completeSets > 0 {
		sharesForSale := math.Min(sellingIncrement, completeSets)
		// Since there are len(outcomes) + 1 different ways to sell the shares back into the market
		// try them each to observe their profitability.

		// Option 1: Sell all shares individually using the bids for each respective outcome.
		// Option 2: Sell one share individually using the bids in its respective orderbook, and
		//           sell the rest of the shares using the asks in the order book.

		// estimatedProceeds[i] is the proceeds from selling complete sets into the outcomes[i] order book.
		// estimatedProceeds[len(outcomes)] is the proceeds from selling each share individually into their respective order books
		estimatedProceeds := make([]float64, len(books)+1)
		for i := 0; i < len(books); i++ {
			estimatedProceeds[len(books)] += books[i].CloseLongFillOnly(sharesForSale, market, true)
			estimatedProceeds[i] += books[i].CloseLongFillOnly(sharesForSale, market, true)
			estimatedProceeds[i] += books[i].CloseShortFillOnly(sharesForSale, market, true)
		}

		// Determine strategy which yields the most proceeds
		maxProceeds := 0.0
		maxProceedsIndex := 0
		for i := 0; i < len(estimatedProceeds); i++ {
			// If strategies are equally profitable we want to de-prioritize taking from each book (option 1 above), because intuitively taking from each book is asymmetric because it takes only bids, not asks. We already de-prioritize taking from each book because it's the last strategy in estimatedProceeds. However, because `estimatedProceeds[len(books)]` is accrued, it's subject to floating point drift, and so we use isMateriallyGreater() to ensure that we're only selecting last strategy if it's materially more profitable.
			if isMateriallyGreater(estimatedProceeds[i], maxProceeds) {
				maxProceeds = estimatedProceeds[i]
				maxProceedsIndex = i
			}
		}

		// Unable to sell, end incremental sell loop
		if maxProceeds == 0 {
			break
		}

		// Execute most profitable strategy
		proceedsFromSale := 0.0
		if maxProceedsIndex == len(books) {
			for i := 0; i < len(books); i++ {
				proceedsFromSale += books[i].CloseLongFillOnly(sharesForSale, market, false)
			}
		} else {
			proceedsFromSale += books[maxProceedsIndex].CloseLongFillOnly(sharesForSale, market, false)
			proceedsFromSale += books[maxProceedsIndex].CloseShortFillOnly(sharesForSale, market, false)
		}
		totalProceeds += proceedsFromSale
		completeSets -= sharesForSale
	}

	return totalProceeds / allowance.Float64()
}

// isMateriallyGreater returns true if and only if `a` is materially greater than `b`. Ie. false if a < b or they are immaterially similar (for our purposes).
func isMateriallyGreater(a, b float64) bool {
	return (a - b) > 0.001
}
