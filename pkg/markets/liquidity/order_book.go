package liquidity

import (
	"github.com/stateshape/augur-analyzer/pkg/proto/markets"
)

type outcomeOrderBook struct {
	Bids []*markets.LiquidityAtPrice
	Asks []*markets.LiquidityAtPrice
}

func NewOutcomeOrderBook(bids []*markets.LiquidityAtPrice, asks []*markets.LiquidityAtPrice) OutcomeOrderBook {
	bidsCopy := []*markets.LiquidityAtPrice{}
	asksCopy := []*markets.LiquidityAtPrice{}
	for _, bid := range bids {
		bidsCopy = append(bidsCopy, &markets.LiquidityAtPrice{
			Price:  bid.Price,
			Amount: bid.Amount,
		})
	}
	for _, ask := range asks {
		asksCopy = append(asksCopy, &markets.LiquidityAtPrice{
			Price:  ask.Price,
			Amount: ask.Amount,
		})
	}

	return &outcomeOrderBook{
		Bids: bidsCopy,
		Asks: asksCopy,
	}
}

func (oob *outcomeOrderBook) DeepClone() OutcomeOrderBook {
	bidsCopy := []*markets.LiquidityAtPrice{}
	asksCopy := []*markets.LiquidityAtPrice{}
	for _, bid := range oob.Bids {
		bidsCopy = append(bidsCopy, &markets.LiquidityAtPrice{
			Price:  bid.Price,
			Amount: bid.Amount,
		})
	}
	for _, ask := range oob.Asks {
		asksCopy = append(asksCopy, &markets.LiquidityAtPrice{
			Price:  ask.Price,
			Amount: ask.Amount,
		})
	}
	return &outcomeOrderBook{
		Bids: bidsCopy,
		Asks: asksCopy,
	}
}

func (oob *outcomeOrderBook) CloseLongFillOnly(shares float64, market MarketData, dryRun bool) float64 {
	bids, proceeds := oob.TakeBest(oob.Bids, shares, market, dryRun, false)
	if !dryRun {
		oob.Bids = bids
	}
	return proceeds
}

func (oob *outcomeOrderBook) CloseShortFillOnly(shares float64, market MarketData, dryRun bool) float64 {
	asks, proceeds := oob.TakeBest(oob.Asks, shares, market, dryRun, true)
	if !dryRun {
		oob.Asks = asks
	}
	return proceeds
}

func (oob *outcomeOrderBook) NormalizeComplementPrice(price float32, market MarketData, closingShort bool) float64 {
	// Complement
	if closingShort {
		return market.MaxPrice - float64(price)
	}
	return float64(price) - market.MinPrice
}

func (oob *outcomeOrderBook) TakeBest(liquidity []*markets.LiquidityAtPrice, shares float64, market MarketData, dryRun bool, closingShort bool) ([]*markets.LiquidityAtPrice, float64) {
	proceeds := 0.0

	for shares > 0 {
		if len(liquidity) < 1 {
			return liquidity, proceeds
		}

		if float64(liquidity[0].Amount) > shares {
			price := oob.NormalizeComplementPrice(liquidity[0].Price, market, closingShort)
			proceeds += shares * price
			if !dryRun {
				liquidity[0].Amount -= float32(shares)
			}
			shares -= shares
			return liquidity, proceeds
		}

		price := oob.NormalizeComplementPrice(liquidity[0].Price, market, closingShort)
		proceeds += float64(liquidity[0].Amount) * price
		shares -= float64(liquidity[0].Amount)
		if !dryRun {
			liquidity[0].Amount = 0
		}
		liquidity = liquidity[1:]
	}
	return liquidity, proceeds
}
