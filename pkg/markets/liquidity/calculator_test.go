package liquidity

import (
	"math"
	"testing"

	"github.com/stateshape/augur-analyzer/pkg/currency"
	"github.com/stateshape/augur-analyzer/pkg/proto/markets"

	"github.com/stretchr/testify/assert"
)

// floating point precision causes result of GetLiquidityRetentionRatio() to vary by epsilon, so we test that we're within epsilon.
func assertWithinEpsilon(t *testing.T, expected, actual float64) {
	t.Run("float values within epsilon", func(t *testing.T) {
		if !(math.Abs(actual-expected) < 0.0000000001) { // this value chosen via empirical observation
			t.Errorf("not within epsilon: expected: %f, actual %f", expected, actual)
		}
	})
}

func TestCalculatorGetLiquidityRetentionRatio(t *testing.T) {
	book := func() *outcomeOrderBook {
		return &outcomeOrderBook{
			Bids: []*markets.LiquidityAtPrice{},
			Asks: []*markets.LiquidityAtPrice{},
		}
	}
	book1 := func() []*outcomeOrderBook {
		return []*outcomeOrderBook{book()}
	}
	toLap := func(price, amount float32) *markets.LiquidityAtPrice {
		return &markets.LiquidityAtPrice{
			Price:  price,
			Amount: amount,
		}
	}
	type lap struct {
		Price  float32
		Amount float32
	}
	bids := func(oobs []*outcomeOrderBook, bookIndex uint64, laps ...lap) []*outcomeOrderBook {
		for _, l := range laps {
			oobs[bookIndex].Bids = append(oobs[bookIndex].Bids, toLap(l.Price, l.Amount))
		}
		return oobs
	}
	asks := func(oobs []*outcomeOrderBook, bookIndex uint64, laps ...lap) []*outcomeOrderBook {
		for _, l := range laps {
			oobs[bookIndex].Asks = append(oobs[bookIndex].Asks, toLap(l.Price, l.Amount))
		}
		return oobs
	}
	orders := func(oobs []*outcomeOrderBook, bookIndex uint64, bidLaps []lap, askLaps []lap) []*outcomeOrderBook {
		return asks(bids(oobs, bookIndex, bidLaps...), bookIndex, askLaps...)
	}
	// in golang, a type `[]foo` isn't a `[]FooInterface` even if `foo` implements `FooInterface`. This called a lack of "covariant types".
	toI := func(oobs []*outcomeOrderBook) []OutcomeOrderBook {
		oobs2 := make([]OutcomeOrderBook, len(oobs))
		for i := range oobs {
			oobs2[i] = oobs[i]
		}
		return oobs2
	}

	shareSellingIncrement := 0.01
	allowance := currency.Ether(5)

	marketDataForYesNoAndCategoricalMarkets := MarketData{MinPrice: 0.0, MaxPrice: 1.0}
	md := marketDataForYesNoAndCategoricalMarkets // alias for anti-spam

	defaultCompleteSetsForYesNoAndCategoricalMarkets := float32(allowance.Float64())
	cs := defaultCompleteSetsForYesNoAndCategoricalMarkets // alias for anti-spam

	cases := []struct {
		Name                   string
		OutcomeOrderBooks      []*outcomeOrderBook
		Market                 MarketData
		ShareSellingIncrement  float64
		Allowance              currency.Ether
		ExpectedRetentionRatio float64
	}{
		// TODO convert this test case to []*outcomeOrderBook
		// {Name: "Yes/No", OutcomeOrderBooks: []OutcomeOrderBook{NewOutcomeOrderBook([]*markets.LiquidityAtPrice{{Price: .5, Amount: 2}, {Price: .45, Amount: 2}, {Price: .4, Amount: 2}, {Price: .35, Amount: 2}, {Price: .3, Amount: 2}}, []*markets.LiquidityAtPrice{{Price: .6, Amount: 2}, {Price: .65, Amount: 2}, {Price: .7, Amount: 2}, {Price: .75, Amount: 2}, {Price: .8, Amount: 2}})}, Market: MarketData{MinPrice: 0.0, MaxPrice: 1.0}, ShareSellingIncrement: 0.01, Allowance: currency.Ether(5), ExpectedRetentionRatio: 0.82},
		{"Yes/No - perfect liquidity", orders(book1(), 0, []lap{lap{0.5, cs}}, []lap{lap{0.5, cs}}), md, shareSellingIncrement, allowance, 1},
	}

	calculator := NewCalculator()

	testsRun := 0
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			rr := calculator.GetLiquidityRetentionRatio(c.ShareSellingIncrement, c.Allowance, c.Market, toI(c.OutcomeOrderBooks))
			assertWithinEpsilon(t, c.ExpectedRetentionRatio, rr)
			// assert.True(t, withinEpsilon(c.ExpectedRetentionRatio, rr))
			testsRun++
		})
	}
	assert.Equal(t, len(cases), testsRun, "sanity check that all tests ran")
}
