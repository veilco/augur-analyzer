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
		if !(math.Abs(actual-expected) < 0.00001) { // this value chosen via empirical observation
			t.Errorf("not within epsilon: expected: %f, actual %f, delta: %.10f", expected, actual, math.Abs((actual - expected)))
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
	book4 := func() []*outcomeOrderBook {
		return []*outcomeOrderBook{book(), book(), book(), book()}
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

	marketDataForScalarMarkets := MarketData{MinPrice: -100, MaxPrice: 1200.0}
	mdScalar := marketDataForScalarMarkets

	scalarCompleteSets := allowance.Float64() / (mdScalar.MaxPrice - mdScalar.MinPrice)

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
		{Name: "Yes/No", OutcomeOrderBooks: []*outcomeOrderBook{NewOutcomeOrderBook([]*markets.LiquidityAtPrice{{Price: .5, Amount: 2}, {Price: .45, Amount: 2}, {Price: .4, Amount: 2}, {Price: .35, Amount: 2}, {Price: .3, Amount: 2}}, []*markets.LiquidityAtPrice{{Price: .6, Amount: 2}, {Price: .65, Amount: 2}, {Price: .7, Amount: 2}, {Price: .75, Amount: 2}, {Price: .8, Amount: 2}}).(*outcomeOrderBook)}, Market: MarketData{MinPrice: 0.0, MaxPrice: 1.0}, ShareSellingIncrement: 0.01, Allowance: currency.Ether(5), ExpectedRetentionRatio: 0.82},
		{"Yes/No - perfect liquidity #1", orders(book1(), 0, []lap{lap{0.5, cs}}, []lap{lap{0.5, cs}}), md, shareSellingIncrement, allowance, 1},
		{"Scalar - perfect liquidity #1", orders(book1(), 0, []lap{lap{585, cs}}, []lap{lap{585, cs}}), mdScalar, shareSellingIncrement, allowance, 1},
		{"Categorical - perfect liquidity #1", orders(orders(book4(), 1, []lap{lap{0.5, 2.5}}, []lap{lap{0.5, 2.5}}), 2, []lap{lap{0.5, 5}}, []lap{lap{0.5, 5}}), md, shareSellingIncrement, allowance, 1},
		{"Yes/No - perfect liquidity #2", orders(book1(), 0, []lap{lap{0.115, cs * 2}}, []lap{lap{0.115, cs * 1.5}}), md, shareSellingIncrement, allowance, 1},
		{"Scalar - perfect liquidity #2", orders(book1(), 0, []lap{lap{-28, cs}}, []lap{lap{-28, cs * 10}}), mdScalar, shareSellingIncrement, allowance, 1},
		{"Categorical - perfect liquidity #2", orders(orders(book4(), 1, []lap{lap{0.4, 2}}, []lap{lap{0.4, 2}}), 2, []lap{lap{0.6, 3}}, []lap{lap{0.6, 3}}), md, shareSellingIncrement, allowance, 1},
		{"Categorical - perfect liquidity #3", bids(bids(bids(bids(book4(), 0, lap{0.4, 5}), 1, lap{0.2, 5}), 2, lap{0.15, 5}), 3, lap{0.25, 5}), md, shareSellingIncrement, allowance, 1},
		{"Categorical - perfect liquidity #4", bids(bids(bids(bids(book4(), 0, lap{0.45, 4}, lap{0.35, 1}), 1, lap{0.2, 4}, lap{0.15, 1}), 2, lap{0.15, 4}, lap{0.10, 1}), 3, lap{0.25, 4}, lap{0.20, 4}), md, shareSellingIncrement, allowance, 1},
		{"Categorical - perfect liquidity #5", asks(bids(bids(bids(bids(book4(), 0, lap{0.4, 3}), 1, lap{0.2, 3}), 2, lap{0.15, 3}), 3, lap{0.25, 3}, lap{0.2, 2}), 3, lap{0.2, 2}), md, shareSellingIncrement, allowance, 1},
		{"YesNo - no liquidity", book1(), md, shareSellingIncrement, allowance, 0},
		{"Scalar - no liquidity", book1(), mdScalar, shareSellingIncrement, allowance, 0},
		{"Categorical - no liquidity", book4(), md, shareSellingIncrement, allowance, 0},
		{"Yes/No - low liquidity due to wide spread", orders(book1(), 0, []lap{lap{0.2, cs * 10}}, []lap{lap{0.6, cs * 25}}), md, shareSellingIncrement, allowance, 0.6},
		{"Scalar - low liquidity due to wide spread", orders(book1(), 0, []lap{lap{0, cs * 123}}, []lap{lap{500, cs * 25}}), mdScalar, shareSellingIncrement, allowance, 800.0 / 1300}, // ie. share price is 1300, we get 100 from closing long and (1200-500 = 700) from closing short, ratio is 800/1300

		{"Yes/No - low liquidity due to insufficient quantity", orders(book1(), 0, []lap{lap{0.5, cs * 0.25}}, []lap{lap{0.5, cs * 0.5}}), md, shareSellingIncrement, allowance, 0.25 + 0.5*0.25}, // 100% price on 25% of shares, 50% on next 25% of shares, 0% on last 50% of shares (this arithmetic only works because it's at 0.5 price halfway between max and min)

		{"Scalar - low liquidity due to insufficient quantity", orders(book1(), 0, []lap{lap{800, float32(scalarCompleteSets / 3.0)}}, []lap{lap{800, float32(scalarCompleteSets / 3.0 * 2)}}), mdScalar, shareSellingIncrement, allowance, 1.0/3 + 400.0/(1300*3)}, // 100% of price on 33% of shares + (400/1300) on next third of shares
		{"Categorical - low liquidity due to wide spread #1", orders(orders(book4(), 1, []lap{lap{0.4, 2.5}}, []lap{lap{0.5, 2.5}}), 2, []lap{lap{0.4, 5}}, []lap{lap{0.5, 5}}), md, shareSellingIncrement, allowance, 0.9},
		{"Categorical - low liquidity due to wide spread #2", asks(bids(bids(bids(bids(book4(), 0, lap{0.4, 3}), 1, lap{0.2, 3}), 2, lap{0.15, 3}), 3, lap{0.25, 3}, lap{0.2, 2}), 3, lap{0.6, 2}), md, shareSellingIncrement, allowance, 4.2 / 5},
		{"Categorical - low liquidity due to insufficient quantity", orders(orders(book4(), 1, []lap{lap{0.6, 2.5}}, []lap{lap{0.6, 2.5}}), 2, []lap{lap{0.4, 1.5}}, []lap{lap{0.4, 1.5}}), md, shareSellingIncrement, allowance, 0.8},
		{"Categorical - low liquidity due to wide spread and insufficient quantity", asks(bids(bids(bids(bids(book4(), 0, lap{0.4, 3}), 1, lap{0.2, 3}), 2, lap{0.15, 3}), 3, lap{0.25, 3}, lap{0.15, 1}), 3, lap{0.65, 1}), md, shareSellingIncrement, allowance, 0.7},

		/*
			TODO more test cases
			requires multiple orders to sell all
				yesNo
				scalar
			categorical - more complex examples
				strategy to use only one book(s)
				strategy where all outcomes have liquidity for individual sales
				strategy from all books, with 4 outcomes
				strategy from one book+all books
		*/
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
