package markets

import (
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/stateshape/augur-analyzer/pkg/env"
	"github.com/stateshape/augur-analyzer/pkg/proto/markets"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func DebugMarkets(inputMarketsData *MarketsData, outputMarketsData []*markets.Market) {
	// Catch all panics
	defer func() {
		if err := recover(); err != nil {
			logrus.WithFields(logrus.Fields{
				"stack": string(debug.Stack()),
				"error": err,
			}).Errorf("Recovered in `DebugMarkets`")
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
		logrus.WithField("marketId", mid).Warnf("--- Market Debug Information ---")
		inputMarketData, ok := inputMarketsData.ByMarketID[mid]
		if !ok {
			logrus.WithField("marketId", mid).Warnf("Input market data no found for market while debugging")
		} else {
			printInputMarketData(mid, inputMarketData)
		}

		outputMarketData, ok := outputMarketsDataByID[mid]
		if !ok {
			logrus.WithField("marketId", mid).Warnf("Output market data not found for market while debugging")
		} else {
			printOutputMarketData(outputMarketData)
		}
	}
}

func printInputMarketData(id string, data *MarketData) {
	logrus.WithField("marketId", id).Warnf("Input data for market")

	if data.Info != nil {
		logrus.WithField("marketId", id).Warnf("Market Type: %s", data.Info.MarketType)
		logrus.WithField("marketId", id).Warnf("Outstanding Shares: %s", data.Info.OutstandingShares)
		logrus.WithField("marketId", id).Warnf("Category: %s", data.Info.Category)
		logrus.WithField("marketId", id).Warnf("Description: %s", data.Info.Description)
		logrus.WithField("marketId", id).Warnf("Resolution Source: %s", data.Info.ResolutionSource)
		logrus.WithField("marketId", id).Warnf("Num Ticks: %s", data.Info.NumTicks)
		logrus.WithField("marketId", id).Warnf("Tick Size: %s", data.Info.TickSize)
		logrus.WithField("marketId", id).Warnf("Min Price: %s", data.Info.MinPrice)
		logrus.WithField("marketId", id).Warnf("Max Price: %s", data.Info.MaxPrice)
		logrus.WithField("marketId", id).Warnf("Cumulative Scale: %s", data.Info.CumulativeScale)
		for _, outcome := range data.Info.Outcomes {
			logrus.WithFields(logrus.Fields{
				"marketId":           id,
				"outcomeId":          outcome.Id,
				"outcomeVolume":      outcome.Volume,
				"outcomePrice":       outcome.Price,
				"outcomeDescription": outcome.Description,
			}).Warnf("Outcome info")
		}
	} else {
		logrus.WithField("marketId", id).Warnf("MarketData.Info is nil")
	}

	if data.Orders != nil {
		for outcomeID, ordersByOrderType := range data.Orders.OrdersByOrderIdByOrderTypeByOutcome {
			logrus.WithFields(logrus.Fields{
				"marketId":  id,
				"outcomeId": outcomeID,
			}).Warnf("Orders for market outcome")
			if ordersByOrderType.BuyOrdersByOrderId != nil {
				for orderID, order := range ordersByOrderType.BuyOrdersByOrderId.OrdersByOrderId {
					logrus.WithFields(logrus.Fields{
						"marketId":  id,
						"outcomeId": outcomeID,
						"orderId":   orderID,
						"order":     fmt.Sprintf("%+v", *order),
					}).Warnf("Buy order for market outcome")
				}
			} else {
				logrus.WithFields(logrus.Fields{
					"marketId":  id,
					"outcomeId": outcomeID,
				}).Warnf("No buy orders for market outcome")
			}
			if ordersByOrderType.SellOrdersByOrderId != nil {
				for orderID, order := range ordersByOrderType.SellOrdersByOrderId.OrdersByOrderId {
					logrus.WithFields(logrus.Fields{
						"marketId":  id,
						"outcomeId": outcomeID,
						"orderId":   orderID,
						"order":     fmt.Sprintf("%+v", *order),
					}).Warnf("Sell order for market outcome")
				}
			} else {
				logrus.WithFields(logrus.Fields{
					"marketId":  id,
					"outcomeId": outcomeID,
				}).Warnf("No sell orders for market outcome")
			}
		}
	} else {
		logrus.WithField("marketId", id).Warnf("MarketData.Orders is nil")
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
				}).Warnf("Timestamped price with amount for outcome")
			}
		}
	} else {
		logrus.WithField("marketId", id).Warnf("MarketData.PriceHistory is nil")
	}
}

func printOutputMarketData(data *markets.Market) {
	logrus.WithField("marketId", data.Id).Warnf("Output data for market")

	// Best bids
	if data.BestBids == nil {
		logrus.WithField("marketId", data.Id).Warnf("Best bids for market is nil")
	} else {
		for outcomeID, liquidity := range data.BestBids {
			logrus.WithFields(logrus.Fields{
				"marketId":  data.Id,
				"outcomeId": outcomeID,
				"amount":    liquidity.Amount,
				"price":     liquidity.Price,
			}).Warnf("Best bid for market outcome")
		}
	}

	// Best asks
	if data.BestAsks == nil {
		logrus.WithField("marketId", data.Id).Warnf("Best asks for market is nil")
	} else {
		for outcomeID, liquidity := range data.BestAsks {
			logrus.WithFields(logrus.Fields{
				"marketId":  data.Id,
				"outcomeId": outcomeID,
				"amount":    liquidity.Amount,
				"price":     liquidity.Price,
			}).Warnf("Best ask for market outcome")
		}
	}

	// Liquidity at Prices
	if data.Bids == nil || len(data.Bids) == 0 {
		logrus.WithField("marketId", data.Id).Warnf("Best bids for market is nil")
	} else {
		for outcomeID, list := range data.Bids {
			for i, liquidity := range list.LiquidityAtPrice {
				logrus.WithFields(logrus.Fields{
					"index":     i,
					"marketId":  data.Id,
					"outcomeId": outcomeID,
					"amount":    liquidity.Amount,
					"price":     liquidity.Price,
				}).Warnf("Bid liquidity")
			}
		}
	}
	if data.Asks == nil || len(data.Asks) == 0 {
		logrus.WithField("marketId", data.Id).Warnf("Best asks for market is nil")
	} else {
		for outcomeID, list := range data.Asks {
			for i, liquidity := range list.LiquidityAtPrice {
				logrus.WithFields(logrus.Fields{
					"index":     i,
					"marketId":  data.Id,
					"outcomeId": outcomeID,
					"amount":    liquidity.Amount,
					"price":     liquidity.Price,
				}).Warnf("Ask liquidity")
			}
		}
	}

	// Predictions
	if len(data.Predictions) == 0 {
		logrus.WithField("marketId", data.Id).Warnf("No predictions for market")
	} else {
		for _, prediction := range data.Predictions {
			logrus.WithFields(logrus.Fields{
				"marketId":  data.Id,
				"outcomeId": prediction.OutcomeId,
				"percent":   prediction.Percent,
				"value":     prediction.Value,
				"name":      prediction.Name,
			}).Warnf("Prediction for outcome")
		}
	}
}
