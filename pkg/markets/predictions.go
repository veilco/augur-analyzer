package markets

import (
	"sort"
	"strconv"

	"github.com/sirupsen/logrus"
	"github.com/stateshape/augur-analyzer/pkg/proto/augur"
	"github.com/stateshape/augur-analyzer/pkg/proto/markets"
)

type Outcome struct {
	ID          uint64
	Description string
	Volume      float64
	Price       float64
}

type Market struct {
	MinPrice           float64
	MaxPrice           float64
	Volume             float64
	ScalarDenomination string
}

func convertToMarket(m *augur.MarketInfo) (*Market, error) {
	min, err := strconv.ParseFloat(m.MinPrice, 64)
	if err != nil {
		return &Market{}, err
	}
	max, err := strconv.ParseFloat(m.MaxPrice, 64)
	if err != nil {
		return &Market{}, err
	}
	volume, err := strconv.ParseFloat(m.Volume, 64)
	if err != nil {
		return &Market{}, err
	}
	return &Market{
		MinPrice:           min,
		MaxPrice:           max,
		Volume:             volume,
		ScalarDenomination: m.ScalarDenomination,
	}, nil
}

func convertToOutcomes(ois []*augur.OutcomeInfo) ([]*Outcome, error) {
	outcomes := []*Outcome{}
	for _, oi := range ois {
		volume, err := strconv.ParseFloat(oi.Volume, 64)
		if err != nil {
			return []*Outcome{}, err
		}
		price, err := strconv.ParseFloat(oi.Price, 64)
		if err != nil {
			return []*Outcome{}, err
		}
		outcomes = append(outcomes, &Outcome{
			ID:          oi.Id,
			Description: oi.Description,
			Volume:      volume,
			Price:       price,
		})
	}
	return outcomes, nil
}

// For each outcome we want to find the highest buy orders and
// the lowest sell orders
func getBestBids(orders *augur.GetOrdersResponse_OrdersByOrderIdByOrderTypeByOutcome) (map[uint64]*markets.LiquidityAtPrice, error) {
	bestBidsByOutcome := map[uint64]*markets.LiquidityAtPrice{}

	if orders == nil || orders.OrdersByOrderIdByOrderTypeByOutcome == nil {
		return bestBidsByOutcome, nil
	}

	for outcome, ordersByOrderType := range orders.OrdersByOrderIdByOrderTypeByOutcome {
		bestBidsByOutcome[outcome] = &markets.LiquidityAtPrice{
			Price:  0.0,
			Amount: 0.0,
		}
		if ordersByOrderType.BuyOrdersByOrderId == nil || len(ordersByOrderType.BuyOrdersByOrderId.OrdersByOrderId) == 0 {
			continue
		}

		for _, order := range ordersByOrderType.BuyOrdersByOrderId.OrdersByOrderId {
			if order.OrderState != augur.OrderState_OPEN {
				continue
			}
			price, err := strconv.ParseFloat(order.Price, 32)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"orderPrice":           order.Price,
					"orderAmount":          order.Amount,
					"orderId":              order.OrderId,
					"orderTransactionHash": order.TransactionHash,
				}).Errorf("Failed to parse order price from string")
				return map[uint64]*markets.LiquidityAtPrice{}, err
			}
			amount, err := strconv.ParseFloat(order.Amount, 32)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"orderPrice":           order.Price,
					"orderAmount":          order.Amount,
					"orderId":              order.OrderId,
					"orderTransactionHash": order.TransactionHash,
				}).Errorf("Failed to parse order amount from string")
				return map[uint64]*markets.LiquidityAtPrice{}, err
			}

			// Accumulate all liquidity at the highest bid price
			if float64(bestBidsByOutcome[outcome].Price) == price {
				bestBidsByOutcome[outcome].Amount += float32(amount)
			} else if float64(bestBidsByOutcome[outcome].Price) < price {
				bestBidsByOutcome[outcome].Price = float32(price)
				bestBidsByOutcome[outcome].Amount = float32(amount)
			}
		}
	}
	return bestBidsByOutcome, nil
}

func getBestAsks(orders *augur.GetOrdersResponse_OrdersByOrderIdByOrderTypeByOutcome) (map[uint64]*markets.LiquidityAtPrice, error) {
	bestAsksByOutcome := map[uint64]*markets.LiquidityAtPrice{}

	if orders == nil || orders.OrdersByOrderIdByOrderTypeByOutcome == nil {
		return bestAsksByOutcome, nil
	}

	for outcome, ordersByOrderType := range orders.OrdersByOrderIdByOrderTypeByOutcome {
		bestAsksByOutcome[outcome] = &markets.LiquidityAtPrice{
			Price:  0.0,
			Amount: 0.0,
		}
		if ordersByOrderType.SellOrdersByOrderId == nil || len(ordersByOrderType.SellOrdersByOrderId.OrdersByOrderId) == 0 {
			continue
		}

		for _, order := range ordersByOrderType.SellOrdersByOrderId.OrdersByOrderId {
			if order.OrderState != augur.OrderState_OPEN {
				continue
			}
			price, err := strconv.ParseFloat(order.Price, 32)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"orderPrice":           order.Price,
					"orderAmount":          order.Amount,
					"orderId":              order.OrderId,
					"orderTransactionHash": order.TransactionHash,
				}).Errorf("Failed to parse order price from string")
				return map[uint64]*markets.LiquidityAtPrice{}, err
			}
			amount, err := strconv.ParseFloat(order.Amount, 32)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"orderPrice":           order.Price,
					"orderAmount":          order.Amount,
					"orderId":              order.OrderId,
					"orderTransactionHash": order.TransactionHash,
				}).Errorf("Failed to parse order amount from string")
				return map[uint64]*markets.LiquidityAtPrice{}, err
			}

			// Accumulate all liquidity at the highest bid price
			if float64(bestAsksByOutcome[outcome].Price) == price {
				bestAsksByOutcome[outcome].Amount += float32(amount)
			} else if float64(bestAsksByOutcome[outcome].Price) == 0 || float64(bestAsksByOutcome[outcome].Price) > price {
				bestAsksByOutcome[outcome].Price = float32(price)
				bestAsksByOutcome[outcome].Amount = float32(amount)
			}
		}
	}
	return bestAsksByOutcome, nil
}

func getYesNoPredictions(m *Market, outcomes []*Outcome, bestBids, bestAsks map[uint64]*markets.LiquidityAtPrice) []*markets.Prediction {
	// If the market has no volume and no liquidity, do not return predictions
	if m.Volume <= 0.0 && len(bestBids) == 0 && len(bestAsks) == 0 {
		return []*markets.Prediction{}
	}

	if m.Volume > 0.0 { // Derive predicted percent using prices
		no := outcomes[0]
		yes := outcomes[1]

		percent := yes.Price
		if yes.Volume <= 0.0 && no.Volume > 0.0 {
			percent = 1.0 - no.Price
		}

		return []*markets.Prediction{
			{
				Name:      "yes",
				Percent:   float32(percent * 100),
				OutcomeId: 1,
			},
		}
	}

	// Derive predicted percent using liquidity
	bestNoBid, noBidExists := bestBids[0]
	bestNoAsk, noAskExists := bestAsks[0]
	bestYesBid, yesBidExists := bestBids[1]
	bestYesAsk, yesAskExists := bestAsks[1]

	var price float32 = 0.0
	if yesBidExists && yesAskExists {
		// If both YES bids and YES asks, use the weighted avg of the
		// best YES bid and the best YES ask
		price = ((bestYesBid.Price * bestYesBid.Amount) + (bestYesAsk.Price * bestYesAsk.Amount)) / (bestYesBid.Amount + bestYesAsk.Amount)
	} else if yesBidExists && !yesAskExists {
		// If YES bids and no YES asks, use the best YES bid
		price = bestYesBid.Price
	} else if !yesBidExists && yesAskExists {
		// If no YES bids and YES asks, use the best YES ask
		price = bestYesAsk.Price
	} else if noBidExists && noAskExists {
		// If NO bids and NO asks, use 1 - the weighted avg of the
		// best NO bid and the best NO ask
		price = 1 - ((bestNoBid.Price*bestNoBid.Amount)+(bestNoAsk.Price*bestNoAsk.Amount))/(bestNoBid.Amount+bestNoAsk.Amount)
	} else if noBidExists && !noAskExists {
		// If NO bids and no NO asks, use 1 - best NO bid
		price = 1 - bestNoBid.Price
	} else if !noBidExists && noAskExists {
		// IF no NO bids and NO asks, use 1 - best NO ask
		price = 1 - bestNoAsk.Price
	}
	return []*markets.Prediction{
		{
			Name:      "yes",
			Percent:   (100 * price),
			OutcomeId: 1,
		},
	}
}

func getCategoricalPredictions(m *Market, outcomes []*Outcome, bestBids, bestAsks map[uint64]*markets.LiquidityAtPrice) []*markets.Prediction {
	// If market has no volume and no liquidity, do not return predictions
	if m.Volume <= 0.0 && len(bestBids) == 0 && len(bestAsks) == 0 {
		return []*markets.Prediction{}
	}

	if m.Volume > 0 {
		// Sort the outcomes so that the prediction are sent to client
		// in order of most probable to least probable
		sort.Slice(outcomes, func(i, j int) bool {
			return outcomes[i].Price > outcomes[j].Price
		})
		predictions := []*markets.Prediction{}
		for _, o := range outcomes {
			predictions = append(predictions, &markets.Prediction{
				Name:      o.Description,
				Percent:   float32(o.Price * 100),
				OutcomeId: o.ID,
			})
		}
		return predictions
	}

	// Derive prediction using liquidity
	liquidityPricedOutcomes := []*Outcome{}
	for _, o := range outcomes {
		liquidityPricedOutcomes = append(liquidityPricedOutcomes, &Outcome{
			ID:          o.ID,
			Description: o.Description,
			Volume:      o.Volume,
			Price: func() float64 {
				bestBid, bestBidExists := bestBids[o.ID]
				bestAsk, bestAskExists := bestAsks[o.ID]

				// If the outcome has no bids and no asks, the approximate prediction is 0
				if !bestBidExists && !bestAskExists {
					return 0.0
				}
				// If the outcome has bids and asks, use the weighted avg of
				// the best bid and the best ask
				if bestBidExists && bestAskExists {
					return float64(((bestBid.Price * bestBid.Amount) + (bestAsk.Price * bestAsk.Amount)) /
						(bestBid.Amount + bestAsk.Amount))
				}
				// If the outcome has bids and no asks, use the best bid
				if bestBidExists {
					return float64(bestBid.Price)
				}
				// If the outcome has no bids and has asks, use the best ask
				return float64(bestAsk.Price)
			}(),
		})
	}

	// Sort outcomes by best approximate prediction and generate Prediction types
	sort.Slice(liquidityPricedOutcomes, func(i, j int) bool {
		return outcomes[i].Price > outcomes[j].Price
	})
	predictions := []*markets.Prediction{}
	for _, o := range outcomes {
		predictions = append(predictions, &markets.Prediction{
			Name:      o.Description,
			Percent:   float32(o.Price * 100),
			OutcomeId: o.ID,
		})
	}
	return predictions
}

func getScalarPredictions(m *Market, outcomes []*Outcome, bestBids, bestAsks map[uint64]*markets.LiquidityAtPrice) []*markets.Prediction {
	if len(outcomes) != 2 {
		logrus.WithField("outcomeInfos", outcomes).Errorf("`getScalarPredictions` was called without 2 `OutcomeInfo` arguments")
		return []*markets.Prediction{}
	}

	lower := outcomes[0]
	upper := outcomes[1]

	// If the market has no volume and no liquidity, do not return predictions
	if m.Volume <= 0.0 && len(bestBids) == 0 && len(bestAsks) == 0 {
		return []*markets.Prediction{}
	}

	if m.Volume > 0 { // Derive predicted scalar value using prices
		value := upper.Price
		if upper.Volume <= 0.0 && lower.Volume > 0.0 {
			value = m.MaxPrice + m.MinPrice - lower.Price
		}

		return []*markets.Prediction{
			{
				Name:      "",
				Value:     float32(value),
				OutcomeId: 1,
			},
		}
	}

	// Derive predicted scalar value using liquidity
	bestLowerBid, lowerBidExists := bestBids[0]
	bestLowerAsk, lowerAskExists := bestAsks[0]
	bestUpperBid, upperBidExists := bestBids[1]
	bestUpperAsk, upperAskExists := bestAsks[1]

	var value float32 = 0.0
	if upperBidExists && upperAskExists {
		// If UPPER bids and UPPER asks, use the weighted average
		// of the best UPPER bid and the best UPPER ask
		value = ((bestUpperBid.Price * bestUpperAsk.Amount) + (bestUpperAsk.Price * bestUpperAsk.Amount)) / (bestUpperBid.Amount + bestUpperAsk.Amount)
	} else if upperBidExists && !upperAskExists {
		// If UPPER bids and no UPPER asks, use the best UPPER bid
		value = bestUpperBid.Price
	} else if !upperBidExists && upperAskExists {
		// If no UPPER bids and UPPER asks, use the best UPPER ask
		value = bestUpperAsk.Price
	} else if lowerBidExists && lowerAskExists {
		// If LOWER bids and LOWER asks, use
		// Market.MaxPrice + Market.MinPrice - (weighted avg of best LOWER bid and best LOWER ask)
		value = float32(m.MaxPrice+m.MinPrice) - ((bestLowerBid.Price*bestLowerBid.Amount)+(bestLowerAsk.Price*bestLowerAsk.Price*bestLowerAsk.Amount))/(bestLowerBid.Amount+bestLowerAsk.Amount)
	} else if lowerBidExists && !lowerAskExists {
		// If LOWER bids and no LOWER asks, use
		// Market.MaxPrice + Market.MinPrice - (best LOWER bid)
		value = float32(m.MaxPrice+m.MinPrice) - bestLowerBid.Price
	} else if !lowerBidExists && lowerAskExists {
		// If no LOWER bids and LOWER asks, use
		// Market.MaxPrice + Market.MinPrice - (best LOWER ask)
		value = float32(m.MaxPrice+m.MinPrice) - bestLowerAsk.Price
	}

	return []*markets.Prediction{
		{
			Name:      "",
			Value:     float32(value),
			OutcomeId: 1,
		},
	}
}
