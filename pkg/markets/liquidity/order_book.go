package liquidity

import "github.com/stateshape/augur-analyzer/pkg/proto/markets"

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

func (oob *outcomeOrderBook) TakeBids(maxSharesToTake float64, opts TakeOptions) (proceeds float64) {
	bids, proceeds := oob.takeBest(oob.Bids, maxSharesToTake, opts)
	if !opts.DryRun {
		oob.Bids = bids
	}
	return proceeds
}

func (oob *outcomeOrderBook) TakeAsks(maxSharesToTake float64, opts TakeOptions) (proceeds float64) {
	asks, proceeds := oob.takeBest(oob.Asks, maxSharesToTake, opts)
	if !opts.DryRun {
		oob.Asks = asks
	}
	return proceeds
}

func (oob *outcomeOrderBook) takeBest(liquidity []*markets.LiquidityAtPrice, maxSharesToTake float64, opts TakeOptions) ([]*markets.LiquidityAtPrice, float64) {
	proceeds := 0.0
	for maxSharesToTake > 0 {
		if len(liquidity) < 1 {
			return liquidity, proceeds
		}
		if float64(liquidity[0].Amount) > maxSharesToTake {
			if !opts.DryRun {
				liquidity[0].Amount -= float32(maxSharesToTake)
			}
			proceeds += maxSharesToTake * float64(liquidity[0].Price)
			return liquidity, proceeds
		}
		proceeds += float64(liquidity[0].Amount * liquidity[0].Price)
		maxSharesToTake -= float64(liquidity[0].Amount)
		if !opts.DryRun {
			liquidity[0].Amount = 0
		}
		liquidity = liquidity[1:]
	}
	return liquidity, proceeds
}
