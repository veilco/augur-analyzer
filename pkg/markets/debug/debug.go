package debug

import (
	"strings"

	"github.com/stateshape/augur-analyzer/pkg/env"
	marketsx "github.com/stateshape/augur-analyzer/pkg/markets"
	"github.com/stateshape/augur-analyzer/pkg/proto/markets"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func DebugMarkets(inputMarketsData *marketsx.MarketsData, outputMarketsData []*markets.Market) {
	// Catch all panics
	defer func() {
		if err := recover(); err != nil {
			logrus.Errorf("Recovered in `DebugMarkets: %#v", err)
		}
	}()

	marketsCSV := viper.GetString(env.DebugMarkets)
	if len(marketsCSV) == 0 {
		return
	}

	marketIDs := strings.Split(marketsCSV, ",")

	// Index of markets being debugged
	mids := map[string]struct{}{}
	for _, mid := range marketIDs {
		mids[mid] = struct{}{}
	}

	// Index into markets by id
	outputMarketsDataByID := map[string]*markets.Market{}
	for _, m := range outputMarketsData {
		if _, ok := mids[m.Id]; ok {
			outputMarketsDataByID[m.Id] = m
		}
	}

	for mid, _ := range mids {
		inputMarketData, ok := inputMarketsData.ByMarketID[mid]
		if !ok {
			logrus.WithField("marketId", mid).Debugf("Input market data no found for market while debugging")
		} else {
			printInputMarketData(mid, inputMarketData)
		}

		outputMarketData, ok := outputMarketsDataByID[mid]
		if !ok {
			logrus.WithField("marketId", mid).Debugf("Output market data not found for market while debugging")
		} else {
			printOutputMarketData(outputMarketData)
		}
	}
}

func printInputMarketData(id string, data *marketsx.MarketData) {
	logrus.WithField("marketId", id).Debugf("Input data for market")

	if data.Info != nil {
		logrus.WithField("marketId", id).Debugf("Outstanding Shares: %s", data.Info.OutstandingShares)
		logrus.WithField("marketId", id).Debugf("Category: %s", data.Info.Category)
		logrus.WithField("marketId", id).Debugf("Description: %s", data.Info.Description)
		logrus.WithField("marketId", id).Debugf("Resolution Source: %s", data.Info.ResolutionSource)
		logrus.WithField("marketId", id).Debugf("Num Ticks: %s", data.Info.NumTicks)
		logrus.WithField("marketId", id).Debugf("Tick Size: %s", data.Info.TickSize)
		for _, outcome := range data.Info.Outcomes {
			logrus.WithFields(logrus.Fields{
				"marketId":           id,
				"outcomeId":          outcome.Id,
				"outcomeVolume":      outcome.Volume,
				"outcomePrice":       outcome.Price,
				"outcomeDescription": outcome.Description,
			}).Debugf("Outcome info")
		}
	} else {
		logrus.WithField("marketId", id).Debugf("MarketData.Info is nil")
	}

	if data.Orders != nil {
		for outcomeID, ordersByOrderType := range data.Orders.OrdersByOrderIdByOrderTypeByOutcome {
			logrus.WithFields(logrus.Fields{
				"marketId":  id,
				"outcomeId": outcomeID,
			}).Debugf("Orders for market outcome")
			for orderID, order := range ordersByOrderType.BuyOrdersByOrderId.OrdersByOrderId {
				logrus.WithFields(logrus.Fields{
					"marketId":  id,
					"outcomeId": outcomeID,
					"orderId":   orderID,
					"order":     *order,
				}).Debugf("Buy order for market outcome")
			}
			for orderID, order := range ordersByOrderType.SellOrdersByOrderId.OrdersByOrderId {
				logrus.WithFields(logrus.Fields{
					"marketId":  id,
					"outcomeId": outcomeID,
					"orderId":   orderID,
					"order":     *order,
				}).Debugf("Sell order for market outcome")
			}
		}
	} else {
		logrus.WithField("marketId", id).Debugf("MarketData.Orders is nil")
	}

	if data.PriceHistory != nil {
		for outcomeID, timestampedPrices := range data.PriceHistory.MarketPriceHistory.TimestampedPriceAmountByOutcome {
			for _, timestampedPrice := range timestampedPrices.TimestampedPriceAmounts {
				logrus.WithFields(logrus.Fields{
					"marketId":  id,
					"outcomeId": outcomeID,
					"price":     timestampedPrice.Price,
					"amount":    timestampedPrice.Amount,
					"timestamp": timestampedPrice.Timestamp,
				}).Debugf("Timestamped price with amount for outcome")
			}
		}
	} else {
		logrus.WithField("marketId", id).Debugf("MarketData.PriceHistory is nil")
	}
}

func printOutputMarketData(data *markets.Market) {
	logrus.WithField("marketId", data.Id).Debugf("Output data for market")

	// Best bids
	if data.BestBids == nil {
		logrus.WithField("marketId", data.Id).Debugf("Best bids for market is nil")
	} else {
		for outcomeID, liquidity := range data.BestBids {
			logrus.WithFields(logrus.Fields{
				"marketId":  data.Id,
				"outcomeId": outcomeID,
				"amount":    liquidity.Amount,
				"price":     liquidity.Price,
			}).Debugf("Best bid for market outcome")
		}
	}

	// Best asks
	if data.BestAsks == nil {
		logrus.WithField("marketId", data.Id).Debugf("Best asks for market is nil")
	} else {
		for outcomeID, liquidity := range data.BestAsks {
			logrus.WithFields(logrus.Fields{
				"marketId":  data.Id,
				"outcomeId": outcomeID,
				"amount":    liquidity.Amount,
				"price":     liquidity.Price,
			}).Debugf("Best ask for market outcome")
		}
	}

	// Predictions
	if len(data.Predictions) == 0 {
		logrus.WithField("marketId", data.Id).Debugf("No predictions for market")
	} else {
		for _, prediction := range data.Predictions {
			logrus.WithFields(logrus.Fields{
				"marketId":  data.Id,
				"outcomeId": prediction.OutcomeId,
				"percent":   prediction.Percent,
				"value":     prediction.Value,
				"name":      prediction.Name,
			}).Debugf("Prediction for outcome")
		}
	}
}
