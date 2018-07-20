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
	return &Market{
		MinPrice:           min,
		MaxPrice:           max,
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
func getBidAskForOutcomes(orders *augur.GetOrdersResponse_OrdersByOrderIdByOrderTypeByOutcome) (map[uint64]*markets.BidAsk, error) {
	bidasks := map[uint64]*markets.BidAsk{}
	for outcome, ordersByOrderType := range orders.OrdersByOrderIdByOrderTypeByOutcome {
		// If there are no buy and no sell orders then the bidask for the outcome should be nil
		if len(ordersByOrderType.BuyOrdersByOrderId.OrdersByOrderId) == 0 || len(ordersByOrderType.SellOrdersByOrderId.OrdersByOrderId) == 0 {
			bidasks[outcome] = nil
			continue
		}

		// Zero out all the values
		bidasks[outcome] = &markets.BidAsk{}

		// Find best buy order
		for _, order := range ordersByOrderType.BuyOrdersByOrderId.OrdersByOrderId {
			if order.OrderState != augur.OrderState_OPEN {
				continue
			}
			price, err := strconv.ParseFloat(order.Price, 64)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"orderPrice":           order.Price,
					"orderAmount":          order.Amount,
					"orderId":              order.OrderId,
					"orderTransactionHash": order.TransactionHash,
				}).Errorf("Failed to parse order price from string")
				return map[uint64]*markets.BidAsk{}, err
			}
			amount, err := strconv.ParseFloat(order.Amount, 64)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"orderPrice":           order.Price,
					"orderAmount":          order.Amount,
					"orderId":              order.OrderId,
					"orderTransactionHash": order.TransactionHash,
				}).Errorf("Failed to parse order amount from string")
				return map[uint64]*markets.BidAsk{}, err
			}
			if float64(bidasks[outcome].BestBid) < price {
				bidasks[outcome].BestBid = float32(price)
				bidasks[outcome].BestBidQuantity = float32(amount)
			}
		}

		// Find best sell order
		for _, order := range ordersByOrderType.SellOrdersByOrderId.OrdersByOrderId {
			if order.OrderState != augur.OrderState_OPEN {
				continue
			}
			price, err := strconv.ParseFloat(order.Price, 64)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"orderPrice":           order.Price,
					"orderAmount":          order.Amount,
					"orderId":              order.OrderId,
					"orderTransactionHash": order.TransactionHash,
				}).Errorf("Failed to parse order price from string")
				return map[uint64]*markets.BidAsk{}, err
			}
			amount, err := strconv.ParseFloat(order.Amount, 64)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"orderPrice":           order.Price,
					"orderAmount":          order.Amount,
					"orderId":              order.OrderId,
					"orderTransactionHash": order.TransactionHash,
				}).Errorf("Failed to parse order price from string")
				return map[uint64]*markets.BidAsk{}, err
			}
			if float64(bidasks[outcome].BestAsk) == 0.0 || float64(bidasks[outcome].BestAsk) > price {
				bidasks[outcome].BestAsk = float32(price)
				bidasks[outcome].BestAskQuantity = float32(amount)
			}
		}
	}
	return bidasks, nil
}

func getYesNoPredictions(m *Market, outcomes []*Outcome, orders *augur.GetOrdersResponse_OrdersByOrderIdByOrderTypeByOutcome) ([]*markets.Prediction, error) {
	bidasks, err := getBidAskForOutcomes(orders)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to generate bid ask for outcomes of yesNo market")
		return []*markets.Prediction{}, err
	}

	no := outcomes[0]
	yes := outcomes[1]
	yesBidAsk := bidasks[1]

	// Derive predicted percent
	percent := yes.Price
	if yes.Volume <= 0.0 && no.Volume > 0.0 {
		percent = 1.0 - no.Price
	}

	return []*markets.Prediction{
		{
			Name:    "yes",
			Percent: float32(percent * 100),
			BidAsk:  yesBidAsk,
		},
	}, nil
}

func getCategoricalPredictions(m *Market, outcomes []*Outcome, orders *augur.GetOrdersResponse_OrdersByOrderIdByOrderTypeByOutcome) ([]*markets.Prediction, error) {
	bidAsksByOutcomes, err := getBidAskForOutcomes(orders)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to generate bid ask for outcomes of categorical market")
		return []*markets.Prediction{}, err
	}

	// Sort the outcomes so that the prediction are sent to client
	// in order of most probable to least probable
	sort.Slice(outcomes, func(i, j int) bool {
		return outcomes[i].Price > outcomes[j].Price
	})
	predictions := []*markets.Prediction{}
	for _, o := range outcomes {
		predictions = append(predictions, &markets.Prediction{
			Name:    o.Description,
			Percent: float32(o.Price * 100),
			BidAsk:  bidAsksByOutcomes[o.ID],
		})
	}
	return predictions, nil
}

func getScalarPredictions(m *Market, outcomes []*Outcome, orders *augur.GetOrdersResponse_OrdersByOrderIdByOrderTypeByOutcome) ([]*markets.Prediction, error) {
	if len(outcomes) != 2 {
		logrus.WithField("outcomeInfos", outcomes).Errorf("`getScalarPredictions` was called without 2 `OutcomeInfo` arguments")
		return []*markets.Prediction{}, nil
	}

	bidAskByOutcome, err := getBidAskForOutcomes(orders)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to generate bid ask for outcomes of scalar market")
		return []*markets.Prediction{}, err
	}

	lower := outcomes[0]
	upper := outcomes[1]
	upperBidAsk := bidAskByOutcome[1]

	// Derive predicted scalar value
	value := upper.Price
	if upper.Volume <= 0.0 && lower.Volume > 0.0 {
		value = m.MaxPrice + m.MinPrice - lower.Price
	}

	return []*markets.Prediction{
		{
			Name:   "",
			Value:  float32(value),
			BidAsk: upperBidAsk,
		},
	}, nil
}
