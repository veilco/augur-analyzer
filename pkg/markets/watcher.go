package markets

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/stateshape/augur-analyzer/pkg/env"
	"github.com/stateshape/augur-analyzer/pkg/gcloud"
	"github.com/stateshape/augur-analyzer/pkg/pricing"
	"github.com/stateshape/augur-analyzer/pkg/proto/augur"
	"github.com/stateshape/augur-analyzer/pkg/proto/markets"

	"cloud.google.com/go/storage"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	MarketTypeYesNo       = "yesNo"
	MarketTypeScalar      = "scalar"
	MarketTypeCategorical = "categorical"
)

type Watcher struct {
	PricingAPI pricing.PricingClient
	Web3API    *ethclient.Client
	AugurAPI   augur.MarketsApiClient
	Writer     *Writer
}

type MarketsData struct {
	ByMarketID    map[string]*MarketData
	ExchangeRates *ExchangeRates
}

type MarketData struct {
	Info         *augur.MarketInfo
	Orders       *augur.GetOrdersResponse_OrdersByOrderIdByOrderTypeByOutcome
	PriceHistory *augur.GetMarketPriceHistoryResponse
}

type ExchangeRates struct {
	ETHUSD float64
	BTCETH float64
}

func NewWatcher(pricingAPI pricing.PricingClient, web3API *ethclient.Client, augurAPI augur.MarketsApiClient, storageAPI *storage.Client) *Watcher {
	return &Watcher{pricingAPI, web3API, augurAPI, &Writer{
		Bucket:           viper.GetString(env.GCloudStorageBucket),
		GCloudStorageAPI: storageAPI,
	}}
}

func (w *Watcher) Watch() {
	initial := true
	for {
		if !initial {
			<-time.After(time.Second * 2) // Avoid looping incessantly
		} else {
			initial = false
		}
		logrus.Infof("Starting a new block header subscription")
		if err := w.process(); err != nil {
			logrus.WithError(err).Errorf("Processing new block headers failed.")
			continue
		}
	}
}

func (w *Watcher) process() error {
	var lastProcessedBlockNumber *big.Int
	for {
		time.Sleep(time.Second * 20)
		header, err := w.Web3API.HeaderByNumber(context.TODO(), nil)
		if err != nil {
			logrus.WithError(err).Errorf("Failed to get latest block header")
			continue
		}
		if lastProcessedBlockNumber != nil && (header.Number == nil || header.Number.Cmp(lastProcessedBlockNumber) <= 0) {
			continue
		}

		logrus.WithField("block", header.Number.String()).Info("Processing new block")

		// Query markets
		// Use a limit and loop until the response is empty
		getMarketsResponse, err := w.AugurAPI.GetMarkets(context.TODO(), &augur.GetMarketsRequest{
			Universe: viper.GetString(env.AugurRootUniverse),
		})
		if err != nil {
			logrus.WithError(err).WithField("block", header.Number.String()).
				Errorf("Call to augur-node `GetMarkets` failed")
			logrus.WithError(err).Errorf("Call to augur-node `GetMarkets` failed")
			continue
		}

		marketAddressesUnfiltered := getMarketsResponse.MarketAddresses

		// Filter out blacklist here
		marketAddresses := []string{}
		for _, address := range marketAddressesUnfiltered {
			if _, ok := blacklist[address]; !ok {
				marketAddresses = append(marketAddresses, address)
				continue
			}
			logrus.WithFields(logrus.Fields{
				"address": address,
			}).Infof("Skipping blacklisted market")
		}

		// Accumulate all the market data from the augur index
		marketsData, err := w.getMarketsData(context.TODO(), marketAddresses)
		if err != nil {
			logrus.WithError(err).Errorf("Failed to gather market data")
			continue
		}

		m := []*markets.Market{}
		for _, md := range marketsData.ByMarketID {
			market, err := translateMarketInfoToMarket(md, marketsData.ExchangeRates.ETHUSD, marketsData.ExchangeRates.BTCETH)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"block":         header.Number.String(),
					"marketAddress": md.Info.Id,
				}).WithError(err).Errorf("Failed to translate a market info into a market")

				// Better to have a subset of the markets
				// included in the summary for this block
				// instead of none, so continue
				continue
			}
			m = append(m, market)
		}

		// Order the markets from biggest to smallest
		sort.Slice(m, func(i, j int) bool {
			return m[i].MarketCapitalization.Eth > m[j].MarketCapitalization.Eth
		})

		summary := &markets.MarketsSummary{
			Block:                      header.Number.Uint64(),
			TotalMarkets:               uint64(len(m)),
			TotalMarketsCapitalization: deriveTotalMarketsCapitalization(m),
			Markets:                    m,
			GenerationTime:             uint64(time.Now().Unix()),
		}
		if err := w.Writer.WriteMarketsSummary(summary); err != nil {
			logrus.WithError(err).Errorf("Failed to write markets summary to GCloud storage")
			continue
		}

		logrus.Infof("Successfully uploaded markets summary for block #%s", header.Number.String())
		lastProcessedBlockNumber = header.Number

		go DebugMarkets(marketsData, m)

		// Write snapshot async since it is not mission critical
		go func() {
			snapshot := &markets.MarketsSnapshot{
				MarketsSummary: summary,
				MarketInfos:    mapMarketInfos(marketsData),
			}
			if err := w.Writer.WriteMarketsSnapshot(snapshot); err != nil {
				logrus.WithError(err).Errorf("Failed to write markets snapshot to GCloud storage")
				return
			}
			logrus.Infof("Successfully uploaded markets snapshot for block #%s", header.Number.String())

		}()
		go func() {
			details := constructMarketDetails(m, marketsData)
			// Write `gcloud.MaxIdleConnsPerHost` details at a time

			// Create a channel of work channels, and load all of the work channels in it
			workers := make(chan chan *markets.MarketDetail, gcloud.MaxIdleConnsPerHost)
			for i := 0; i < gcloud.MaxIdleConnsPerHost; i++ {
				// Create work channel and goroutine for each worker
				worker := make(chan *markets.MarketDetail)
				go func() {
					for detail := range worker {
						if err := w.Writer.WriteMarketDetail(detail); err != nil {
							logrus.WithField("marketId", detail.MarketId).WithError(err).Errorf("Failed to write market detail to GCloud storage")
						}
						workers <- worker
					}
				}()
				workers <- worker
			}

			// Assign work
			for _, detail := range details {
				// Get an available worker and send it work
				<-workers <- detail
			}

			// Close all the worker channels
			for i := 0; i < gcloud.MaxIdleConnsPerHost; i++ {
				close(<-workers)
			}
			close(workers)

			logrus.Infof("Successfully uploaded market detail objects for block #%s", header.Number.String())
		}()

		logrus.WithField("blockNumber", header.Number.String()).Infof("Finished processing block")
	}
}

func constructMarketDetails(ms []*markets.Market, msd *MarketsData) []*markets.MarketDetail {
	details := []*markets.MarketDetail{}
	for _, market := range ms {
		detail := &markets.MarketDetail{
			MarketId:      market.Id,
			MarketSummary: market,
		}
		if md, ok := msd.ByMarketID[market.Id]; ok {
			detail.MarketInfo = mapMarketInfo(md.Info)
		}
		details = append(details, detail)
	}
	return details
}

func deriveTotalMarketsCapitalization(ms []*markets.Market) *markets.Price {
	price := &markets.Price{}
	for _, m := range ms {
		price.Eth += m.MarketCapitalization.Eth
		price.Usd += m.MarketCapitalization.Usd
		price.Btc += m.MarketCapitalization.Btc
	}
	return price
}

func translateMarketInfoToMarket(md *MarketData, ethusd, btceth float64) (*markets.Market, error) {
	if md.Info == nil {
		return nil, fmt.Errorf("`translateMarketInfoToMarket` required a non nil MarketInfo as an argument")
	}

	marketCapitalization, err := translateMarketInfoToMarketCapitalization(md.Info, ethusd, btceth)
	if err != nil {
		logrus.WithError(err).
			WithField("marketInfo", *md.Info).
			Errorf("Failed to translate market info into market capitalization")
		return nil, err
	}
	bestBids, err := getBestBids(md.Orders)
	if err != nil {
		logrus.WithError(err).
			WithField("marketInfo", *md.Info).
			Errorf("Failed to get best bids")
		return nil, err
	}
	bestAsks, err := getBestAsks(md.Orders)
	if err != nil {
		logrus.WithError(err).
			WithField("marketInfo", *md.Info).
			Errorf("Failed to get best asks")
		return nil, err
	}
	predictions, err := getPredictions(md.Info, md.Orders, bestBids, bestAsks)
	if err != nil {
		logrus.WithError(err).
			WithField("marketInfo", md.Info).
			Errorf("Failed to translate market info into predictions")
		return nil, err
	}
	marketType, err := getMarketType(md.Info)
	if err != nil {
		logrus.WithError(err).
			WithField("marketInfo", *md.Info).
			Errorf("Failed to get market type")
		return nil, err
	}

	_, featured := featuredlist[md.Info.Id]

	return &markets.Market{
		Id:                   md.Info.Id,
		MarketType:           marketType,
		Name:                 md.Info.Description,
		CommentCount:         0,
		MarketCapitalization: marketCapitalization,
		EndDate:              md.Info.EndTime,
		Predictions:          predictions,
		Author:               md.Info.Author,
		CreationTime:         md.Info.CreationTime,
		CreationBlock:        md.Info.CreationBlock,
		ResolutionSource:     md.Info.ResolutionSource,
		Details:              md.Info.Details,
		Tags:                 md.Info.Tags,
		IsFeatured:           featured,
		Category:             md.Info.Category,
		LastTradeTime:        getLastTradeTimeFromPriceHistory(md.PriceHistory.MarketPriceHistory),
		BestBids:             bestBids,
		BestAsks:             bestAsks,
	}, nil

}

func translateMarketInfoToMarketCapitalization(info *augur.MarketInfo, ethusd, btceth float64) (*markets.Price, error) {
	outstandingShares, err := strconv.ParseFloat(info.OutstandingShares, 64)
	if err != nil {
		logrus.WithError(err).WithField("outstandingShares", info.OutstandingShares).Errorf("Failed to parse outstanding share")
		return nil, err
	}
	if outstandingShares == 0.0 {
		return &markets.Price{
			Eth: 0,
			Usd: 0,
			Btc: 0,
		}, nil
	}
	// TODO: ensure downsizing float will not result in lost information
	return &markets.Price{
		Eth: float32(outstandingShares),
		Usd: float32((outstandingShares * ethusd)),
		Btc: float32((outstandingShares / btceth)),
	}, nil
}

func getMarketType(info *augur.MarketInfo) (markets.MarketType, error) {
	switch strings.ToLower(info.MarketType) {
	case "yesno":
		return markets.MarketType_YESNO, nil
	case "categorical":
		return markets.MarketType_CATEGORICAL, nil
	case "scalar":
		return markets.MarketType_SCALAR, nil
	}
	return 0, fmt.Errorf("Failed to parse market type: %s", info.MarketType)
}

func getPredictions(info *augur.MarketInfo, orders *augur.GetOrdersResponse_OrdersByOrderIdByOrderTypeByOutcome, bestBids, bestAsks map[uint64]*markets.LiquidityAtPrice) ([]*markets.Prediction, error) {
	if info == nil {
		return []*markets.Prediction{}, fmt.Errorf("Non nil MarketInfo required")
	}

	// Translate the market and outcomes to types with numbers
	m, err := convertToMarket(info)
	if err != nil {
		logrus.WithField("marketInfo", info).
			Errorf("Failed to convert `MarketInfo` info a properly typed `Market`")
		return []*markets.Prediction{}, err
	}
	os, err := convertToOutcomes(info.Outcomes)
	if err != nil {
		logrus.WithField("outcomes", info.Outcomes).
			Errorf("Failed to convert `OutcomeInfo` to properly types `Outcome`")
		return []*markets.Prediction{}, err
	}

	// Generate the list of predictions for the market
	predictions := []*markets.Prediction{}
	switch info.MarketType {
	case MarketTypeYesNo:
		predictions = append(predictions, getYesNoPredictions(m, os, bestBids, bestAsks)...)
	case MarketTypeCategorical:
		predictions = append(predictions, getCategoricalPredictions(m, os, bestBids, bestAsks)...)
	case MarketTypeScalar:
		predictions = append(predictions, getScalarPredictions(m, os, bestBids, bestAsks)...)
	}
	return predictions, nil
}

func mapMarketInfos(marketsData *MarketsData) []*markets.MarketInfo {
	mis := []*markets.MarketInfo{}
	for id, md := range marketsData.ByMarketID {
		info := md.Info
		if _, ok := blacklist[id]; ok {
			continue
		}
		mis = append(mis, mapMarketInfo(info))
	}
	return mis
}

func (w *Watcher) getMarketsData(ctx context.Context, marketAddresses []string) (*MarketsData, error) {
	marketDataByID := map[string]*MarketData{}
	chunks := 10
	for x := 0; x < len(marketAddresses); x += chunks {
		limit := x + chunks
		if limit > len(marketAddresses) {
			limit = len(marketAddresses)
		}
		addresses := marketAddresses[x:limit]

		// Market Info
		getMarketsInfoResponse, err := w.AugurAPI.GetMarketsInfo(ctx, &augur.GetMarketsInfoRequest{
			MarketAddresses: addresses,
		})
		if err != nil {
			logrus.WithError(err).Errorf("Call to augur-node `GetMarketsInfo` failed")
			return nil, err
		}
		for _, mi := range getMarketsInfoResponse.MarketInfo {
			marketDataByID[mi.Id] = &MarketData{} // Initialize
			marketDataByID[mi.Id].Info = mi
		}

		// Market Orders
		bulkGetOrdersResponse, err := w.AugurAPI.BulkGetOrders(ctx, &augur.BulkGetOrdersRequest{
			Requests: func() []*augur.GetOrdersRequest {
				requests := []*augur.GetOrdersRequest{}
				for _, address := range addresses {
					requests = append(requests, &augur.GetOrdersRequest{
						Universe:   viper.GetString(env.AugurRootUniverse),
						MarketId:   address,
						OrderState: augur.OrderState_OPEN,
					})
				}
				return requests
			}(),
		})
		if err != nil {
			logrus.WithError(err).Errorf("Call to augur-node `BulkGetOrders` failed")
			return nil, err
		}
		// Assume each response corresponds to one market
		// since that is how the request is structured
		for _, response := range bulkGetOrdersResponse.Responses {
			for marketAddress, orders := range response.Wrapper.OrdersByOrderIdByOrderTypeByOutcomeByMarketId {
				if marketDataByID[marketAddress].Orders != nil {
					logrus.WithField("marketAddress", marketAddress).Warn("Received multiple get orders responses for one market")
				}
				marketDataByID[marketAddress].Orders = orders
			}
		}

		// Market price history
		bulkGetPriceHistoryResponse, err := w.AugurAPI.BulkGetMarketPriceHistory(context.TODO(), &augur.BulkGetMarketPriceHistoryRequest{
			Requests: func() []*augur.GetMarketPriceHistoryRequest {
				requests := []*augur.GetMarketPriceHistoryRequest{}
				for _, address := range addresses {
					requests = append(requests, &augur.GetMarketPriceHistoryRequest{
						MarketId: address,
					})
				}
				return requests
			}(),
		})
		if err != nil {
			logrus.WithError(err).Errorf("Call to augur-node `BulkGetMarketPriceHistory` failed")
			return nil, err
		}
		for marketAddress, priceHistory := range bulkGetPriceHistoryResponse.ResponsesByMarketId {
			marketDataByID[marketAddress].PriceHistory = priceHistory
		}

	}

	// Query exchange rates
	ethusd, err := w.PricingAPI.ETHtoUSD()
	if err != nil {
		logrus.WithError(err).Errorf("Failed to get ETH USD exchange rate")
		return nil, err
	}
	btceth, err := w.PricingAPI.BTCtoETH()
	if err != nil {
		logrus.WithError(err).Errorf("Failed to get BTC ETH exchange rate")
		return nil, err
	}

	return &MarketsData{
		ByMarketID: marketDataByID,
		ExchangeRates: &ExchangeRates{
			ETHUSD: ethusd,
			BTCETH: btceth,
		},
	}, nil
}

func mapMarketInfo(info *augur.MarketInfo) *markets.MarketInfo {
	m := &markets.MarketInfo{
		Id:                        info.Id,
		Universe:                  info.Universe,
		MarketType:                info.MarketType,
		NumOutcomes:               info.NumOutcomes,
		MinPrice:                  info.MinPrice,
		MaxPrice:                  info.MaxPrice,
		CumulativeScale:           info.CumulativeScale,
		Author:                    info.Author,
		CreationTime:              info.CreationTime,
		CreationBlock:             info.CreationBlock,
		CreationFee:               info.CreationFee,
		SettlementFee:             info.SettlementFee,
		ReportingFeeRate:          info.ReportingFeeRate,
		MarketCreatorFeeRate:      info.MarketCreatorFeeRate,
		MarketCreatorFeesBalance:  info.MarketCreatorFeesBalance,
		MarketCreatorMailbox:      info.MarketCreatorMailbox,
		MarketCreatorMailboxOwner: info.MarketCreatorMailboxOwner,
		InitialReportSize:         info.InitialReportSize,
		Category:                  info.Category,
		Tags:                      info.Tags,
		Volume:                    info.Volume,
		OutstandingShares:         info.OutstandingShares,
		FeeWindow:                 info.FeeWindow,
		EndTime:                   info.EndTime,
		FinalizationBlockNumber:   info.FinalizationBlockNumber,
		FinalizationTime:          info.FinalizationTime,
		// ReportingState:            info.ReportingState,
		Forking:               info.Forking,
		NeedsMigration:        info.NeedsMigration,
		Description:           info.Description,
		Details:               info.Details,
		ScalarDenomination:    info.ScalarDenomination,
		DesignatedReporter:    info.DesignatedReporter,
		DesignatedReportStake: info.DesignatedReportStake,
		ResolutionSource:      info.ResolutionSource,
		NumTicks:              info.NumTicks,
		TickSize:              info.TickSize,
		// Consensus:                 info.Consensus,
		// Outcomes:                  info.Outcomes,
	}

	// Map non equal types
	switch info.ReportingState {
	case augur.ReportingState_PRE_REPORTING:
		m.ReportingState = markets.ReportingState_PRE_REPORTING
	case augur.ReportingState_DESIGNATED_REPORTING:
		m.ReportingState = markets.ReportingState_DESIGNATED_REPORTING
	case augur.ReportingState_OPEN_REPORTING:
		m.ReportingState = markets.ReportingState_OPEN_REPORTING
	case augur.ReportingState_CROWDSOURCING_DISPUTE:
		m.ReportingState = markets.ReportingState_CROWDSOURCING_DISPUTE
	case augur.ReportingState_AWAITING_NEXT_WINDOW:
		m.ReportingState = markets.ReportingState_AWAITING_NEXT_WINDOW
	case augur.ReportingState_AWAITING_FINALIZATION:
		m.ReportingState = markets.ReportingState_AWAITING_FINALIZATION
	case augur.ReportingState_FINALIZED:
		m.ReportingState = markets.ReportingState_FINALIZED
	case augur.ReportingState_FORKING:
		m.ReportingState = markets.ReportingState_FORKING
	case augur.ReportingState_AWAITING_NO_REPORT_MIGRATION:
		m.ReportingState = markets.ReportingState_AWAITING_NO_REPORT_MIGRATION
	case augur.ReportingState_AWAITING_FORK_MIGRATION:
		m.ReportingState = markets.ReportingState_AWAITING_FORK_MIGRATION
	default:
		logrus.WithField("reportingState", info.ReportingState).
			Warnf("Unable to assign reporting state for market info, unhandled enum from augur proto")
	}
	if info.Consensus != nil {
		m.Consensus = &markets.NormalizedPayout{
			IsInvalid: info.Consensus.IsInvalid,
			Payout:    info.Consensus.Payout,
		}
	}
	m.Outcomes = []*markets.OutcomeInfo{}
	for _, outcome := range info.Outcomes {
		if outcome == nil {
			continue
		}
		m.Outcomes = append(m.Outcomes, &markets.OutcomeInfo{
			Id:          outcome.Id,
			Volume:      outcome.Volume,
			Price:       outcome.Price,
			Description: outcome.Description,
		})
	}
	return m
}

func getLastTradeTimeFromPriceHistory(priceHistory *augur.MarketPriceHistory) uint64 {
	mostRecent := uint64(0)
	for _, timestampedPrices := range priceHistory.TimestampedPriceAmountByOutcome {
		for _, timestampedPrice := range timestampedPrices.TimestampedPriceAmounts {
			if timestampedPrice.Timestamp > mostRecent {
				mostRecent = timestampedPrice.Timestamp
			}
		}
	}
	return mostRecent
}
