package liquidity_test

import (
	"testing"

	"github.com/stateshape/augur-analyzer/pkg/markets/liquidity"
	"github.com/stateshape/augur-analyzer/pkg/proto/markets"
	"github.com/stretchr/testify/assert"
)

func TestTakeBids(t *testing.T) {
	cases := []struct {
		Name             string
		Bids             []*markets.LiquidityAtPrice
		Asks             []*markets.LiquidityAtPrice
		Shares           float64
		Market           liquidity.MarketData
		ExpectedProceeds float64
	}{
		{
			Name: "Yes/No, even bids",
			Bids: []*markets.LiquidityAtPrice{
				{
					Price:  .9,
					Amount: 2,
				},
				{
					Price:  .8,
					Amount: 2,
				},
				{
					Price:  .7,
					Amount: 2,
				},
				{
					Price:  .6,
					Amount: 2,
				},
				{
					Price:  .5,
					Amount: 2,
				},
			},
			Asks:   []*markets.LiquidityAtPrice{},
			Shares: 10.0,
			Market: liquidity.MarketData{
				MinPrice: 0.0,
				MaxPrice: 1.0,
			},
			ExpectedProceeds: 7.0,
		},
		{
			Name: "Yes/No, uneven bids",
			Bids: []*markets.LiquidityAtPrice{
				{
					Price:  .9,
					Amount: 5,
				},
				{
					Price:  .8,
					Amount: 3,
				},
				{
					Price:  .7,
					Amount: 1,
				},
				{
					Price:  .6,
					Amount: 0.8,
				},
				{
					Price:  .5,
					Amount: 0.2,
				},
			},
			Asks:   []*markets.LiquidityAtPrice{},
			Shares: 10,
			Market: liquidity.MarketData{
				MinPrice: 0.0,
				MaxPrice: 1.0,
			},
			ExpectedProceeds: 8.18,
		},
		{
			Name: "Scalar",
			Bids: []*markets.LiquidityAtPrice{
				{
					Price:  270,
					Amount: .5,
				},
				{
					Price:  260,
					Amount: .3,
				},
				{
					Price:  250,
					Amount: .1,
				},
				{
					Price:  240,
					Amount: 0.8,
				},
				{
					Price:  230,
					Amount: 0.2,
				},
			},
			Asks:   []*markets.LiquidityAtPrice{},
			Shares: 10,
			Market: liquidity.MarketData{
				MinPrice: 200,
				MaxPrice: 300,
			},
			ExpectedProceeds: 96.0,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			oob := liquidity.NewOutcomeOrderBook(c.Bids, c.Asks)
			proceeds := oob.CloseLongFillOnly(c.Shares, c.Market, false)
			assert.Equal(t, c.ExpectedProceeds, proceeds)
		})
	}

}

func TestTakeAsks(t *testing.T) {
	cases := []struct {
		Name             string
		Bids             []*markets.LiquidityAtPrice
		Asks             []*markets.LiquidityAtPrice
		Shares           float64
		Market           liquidity.MarketData
		ExpectedProceeds float64
	}{
		{
			Name: "YesNo",
			Bids: []*markets.LiquidityAtPrice{},
			Asks: []*markets.LiquidityAtPrice{
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
			Shares: 10.0,
			Market: liquidity.MarketData{
				MinPrice: 0.0,
				MaxPrice: 1.0,
			},
			ExpectedProceeds: 3.0,
		},
		{
			Name: "Scalar",
			Bids: []*markets.LiquidityAtPrice{},
			Asks: []*markets.LiquidityAtPrice{
				{
					Price:  230,
					Amount: 2,
				},
				{
					Price:  240,
					Amount: 2,
				},
				{
					Price:  250,
					Amount: 2,
				},
				{
					Price:  260,
					Amount: 2,
				},
				{
					Price:  270,
					Amount: 2,
				},
			},
			Shares: 10.0,
			Market: liquidity.MarketData{
				MinPrice: 200,
				MaxPrice: 300,
			},
			ExpectedProceeds: 500,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			oob := liquidity.NewOutcomeOrderBook(c.Bids, c.Asks)
			proceeds := oob.CloseShortFillOnly(c.Shares, c.Market, true)
			assert.Equal(t, c.ExpectedProceeds, proceeds)
		})
	}

}

func TestTakeBest(t *testing.T) {

}
