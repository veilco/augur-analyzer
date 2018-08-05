package liquidity_test

import (
	"fmt"
	"testing"

	"github.com/stateshape/augur-analyzer/pkg/currency"
	"github.com/stateshape/augur-analyzer/pkg/markets/liquidity"
	"github.com/stateshape/augur-analyzer/pkg/proto/markets"

	"github.com/stretchr/testify/assert"
)

func TestCalculatorGetLiquidityRetentionRatio(t *testing.T) {
	// Create fake order books and verify the calculated retention
	// across different tranches

	calculator := liquidity.NewCalculator()

	// t.Run("No liquidity in single order book", func(t *testing.T) {
	// 	oobs := []liquidity.OutcomeOrderBook{
	// 		liquidity.NewOutcomeOrderBook(
	// 			[]*markets.LiquidityAtPrice{},
	// 			[]*markets.LiquidityAtPrice{},
	// 		),
	// 	}
	// 	market := liquidity.MarketData{
	// 		MinPrice: 0.0,
	// 		MaxPrice: 1.0,
	// 	}
	// 	sharesPerCompleteSet := 0.1
	// 	allowance := currency.Ether(50.0)

	// 	rr := calculator.GetLiquidityRetentionRatio(sharesPerCompleteSet, allowance, market, oobs)
	// 	assert.Equal(t, 0.0, rr)
	// })

	// t.Run("Simple liquidity in single order book", func(t *testing.T) {
	// 	oobs := []liquidity.OutcomeOrderBook{
	// 		liquidity.NewOutcomeOrderBook(
	// 			[]*markets.LiquidityAtPrice{
	// 				{
	// 					Price:  0.49,
	// 					Amount: 50,
	// 				},
	// 			},
	// 			[]*markets.LiquidityAtPrice{
	// 				{
	// 					Price:  0.51,
	// 					Amount: 50,
	// 				},
	// 			},
	// 		),
	// 	}
	// 	market := liquidity.MarketData{
	// 		MinPrice: 0.0,
	// 		MaxPrice: 1.0,
	// 	}
	// 	sharesPerCompleteSet := 0.01
	// 	allowance := currency.Ether(10)

	// 	rr := calculator.GetLiquidityRetentionRatio(sharesPerCompleteSet, allowance, market, oobs)
	// 	assert.Equal(t, .98, rr)
	// })

	// t.Run("Liquidity at multiple price points in single order book", func(t *testing.T) {
	// 	oobs := []liquidity.OutcomeOrderBook{
	// 		liquidity.NewOutcomeOrderBook(
	// 			[]*markets.LiquidityAtPrice{
	// 				{
	// 					Price:  0.6,
	// 					Amount: 4.75,
	// 				},
	// 				{
	// 					Price:  .5,
	// 					Amount: 3,
	// 				},
	// 				{
	// 					Price:  .43,
	// 					Amount: .4,
	// 				},
	// 				{
	// 					Price:  .42,
	// 					Amount: .83,
	// 				},
	// 				{
	// 					Price:  .4,
	// 					Amount: .8,
	// 				},
	// 				{
	// 					Price:  .3,
	// 					Amount: 1,
	// 				},
	// 				{
	// 					Price:  .2,
	// 					Amount: 1,
	// 				},
	// 			},
	// 			[]*markets.LiquidityAtPrice{
	// 				{
	// 					Price:  0.85,
	// 					Amount: 35,
	// 				},
	// 				{
	// 					Price:  .9,
	// 					Amount: 5.53,
	// 				},
	// 			},
	// 		),
	// 	}
	// 	market := liquidity.MarketData{
	// 		MinPrice: 0.0,
	// 		MaxPrice: 1.0,
	// 	}
	// 	sharesPerCompleteSet := 0.01
	// 	allowance := currency.Ether(10)

	// 	rr := calculator.GetLiquidityRetentionRatio(sharesPerCompleteSet, allowance, market, oobs)
	// 	fmt.Printf("Retention Ratio: %f", rr)
	// 	assert.Equal(t, .9, rr)
	// })

	// t.Run("Liquidity at multiple price points in single order book 1", func(t *testing.T) {
	// 	oobs := []liquidity.OutcomeOrderBook{
	// 		liquidity.NewOutcomeOrderBook(
	// 			[]*markets.LiquidityAtPrice{
	// 				{
	// 					Price:  0.45,
	// 					Amount: 1,
	// 				},
	// 				{
	// 					Price:  .4,
	// 					Amount: 1,
	// 				},
	// 				{
	// 					Price:  .3,
	// 					Amount: 1,
	// 				},
	// 				{
	// 					Price:  .2,
	// 					Amount: 1,
	// 				},
	// 				{
	// 					Price:  .1,
	// 					Amount: 1,
	// 				},
	// 			},
	// 			[]*markets.LiquidityAtPrice{
	// 				{
	// 					Price:  0.55,
	// 					Amount: 1,
	// 				},
	// 				{
	// 					Price:  .6,
	// 					Amount: 1,
	// 				},
	// 				{
	// 					Price:  .7,
	// 					Amount: 1,
	// 				},
	// 				{
	// 					Price:  .8,
	// 					Amount: 1,
	// 				},
	// 				{
	// 					Price:  .9,
	// 					Amount: 1,
	// 				},
	// 			},
	// 		),
	// 	}
	// 	market := liquidity.MarketData{
	// 		MinPrice: 0.0,
	// 		MaxPrice: 1.0,
	// 	}
	// 	sharesPerCompleteSet := 0.01
	// 	allowances := []currency.Ether{currency.Ether(.5), currency.Ether(1), currency.Ether(5)}

	// 	for _, allowance := range allowances {
	// 		clones := []liquidity.OutcomeOrderBook{}
	// 		for _, book := range oobs {
	// 			clones = append(clones, book.DeepClone())
	// 		}
	// 		rr := calculator.GetLiquidityRetentionRatio(sharesPerCompleteSet, allowance, market, clones)
	// 		fmt.Printf("Retention Ratio: %f\n", rr)
	// 		assert.Equal(t, .9, rr)
	// 	}
	// })

	t.Run("Liquidity at multiple price points in single order book 1", func(t *testing.T) {
		oobs := []liquidity.OutcomeOrderBook{
			liquidity.NewOutcomeOrderBook(
				[]*markets.LiquidityAtPrice{
					{
						Price:  0.86,
						Amount: 20.28,
					},
					{
						Price:  .2,
						Amount: 33,
					},
				},
				[]*markets.LiquidityAtPrice{
					{
						Price:  0.92,
						Amount: 0.95,
					},
				},
			),
		}
		market := liquidity.MarketData{
			MinPrice: 0.0,
			MaxPrice: 1.0,
		}
		sharesPerCompleteSet := 0.01
		allowances := []currency.Ether{currency.Ether(5), currency.Ether(10), currency.Ether(50)}

		for _, allowance := range allowances {
			clones := []liquidity.OutcomeOrderBook{}
			for _, book := range oobs {
				clones = append(clones, book.DeepClone())
			}
			rr := calculator.GetLiquidityRetentionRatio(sharesPerCompleteSet, allowance, market, clones)
			fmt.Printf("Retention Ratio: %f\n", rr)
			assert.Equal(t, .9, rr)
		}
	})
}
