package liquidity_test

import (
	"testing"

	"github.com/stateshape/augur-analyzer/pkg/currency"
	"github.com/stateshape/augur-analyzer/pkg/markets/liquidity"
	"github.com/stateshape/augur-analyzer/pkg/proto/markets"

	"github.com/stretchr/testify/assert"
)

func TestCalculatorGetLiquidityRetentionRatio(t *testing.T) {
	// Create fake order books and verify the calculated retention
	// across different tranches

	cases := []struct {
		Name                   string
		OutcomeOrderBooks      []liquidity.OutcomeOrderBook
		Market                 liquidity.MarketData
		ShareSellingIncrement  float64
		Allowance              currency.Ether
		ExpectedRetentionRatio float64
	}{
		{
			Name: "Yes/No",
			OutcomeOrderBooks: []liquidity.OutcomeOrderBook{
				liquidity.NewOutcomeOrderBook(
					[]*markets.LiquidityAtPrice{
						{
							Price:  .5,
							Amount: 2,
						},
						{
							Price:  .45,
							Amount: 2,
						},
						{
							Price:  .4,
							Amount: 2,
						},
						{
							Price:  .35,
							Amount: 2,
						},
						{
							Price:  .3,
							Amount: 2,
						},
					},
					[]*markets.LiquidityAtPrice{
						{
							Price:  .6,
							Amount: 2,
						},
						{
							Price:  .65,
							Amount: 2,
						},
						{
							Price:  .7,
							Amount: 2,
						},
						{
							Price:  .75,
							Amount: 2,
						},
						{
							Price:  .8,
							Amount: 2,
						},
					},
				),
			},
			Market: liquidity.MarketData{
				MinPrice: 0.0,
				MaxPrice: 1.0,
			},
			ShareSellingIncrement:  0.01,
			Allowance:              currency.Ether(5),
			ExpectedRetentionRatio: 0.82,
		},
	}

	calculator := liquidity.NewCalculator()

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			rr := calculator.GetLiquidityRetentionRatio(c.ShareSellingIncrement, c.Allowance, c.Market, c.OutcomeOrderBooks)
			assert.Equal(t, rr, c.ExpectedRetentionRatio)
		})
	}
}
