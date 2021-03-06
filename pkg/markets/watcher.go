package markets

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/stateshape/augur-analyzer/pkg/env"
	"github.com/stateshape/augur-analyzer/pkg/gcloud"
	"github.com/stateshape/augur-analyzer/pkg/markets/liquidity"
	"github.com/stateshape/augur-analyzer/pkg/pricing"
	"github.com/stateshape/augur-analyzer/pkg/proto/augur"
	"github.com/stateshape/augur-analyzer/pkg/proto/markets"

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
	PricingAPI          pricing.PricingClient
	Web3API             *ethclient.Client
	AugurAPI            augur.MarketsApiClient
	Writer              *Writer
	LiquidityCalculator liquidity.Calculator
}

type MarketsData struct {
	ByMarketID    map[string]*MarketData
	ExchangeRates *ExchangeRates
}

type MarketData struct {
	Info   *augur.MarketInfo
	Orders *augur.GetOrdersResponse_OrdersByOrderIdByOrderTypeByOutcome
}

type ExchangeRates struct {
	ETHUSD float64
	BTCETH float64
}

func NewWatcher(pricingAPI pricing.PricingClient, web3API *ethclient.Client, augurAPI augur.MarketsApiClient, objectUploader *gcloud.ObjectUploader) *Watcher {
	return &Watcher{pricingAPI, web3API, augurAPI, &Writer{
		Bucket:         viper.GetString(env.GCloudStorageBucket),
		ObjectUploader: objectUploader,
	}, liquidity.NewCalculator()}
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
			market, err := w.translateMarketInfoToMarket(md, marketsData.ExchangeRates.ETHUSD, marketsData.ExchangeRates.BTCETH)
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

		summary := &markets.MarketsSummary{
			Block:                      header.Number.Uint64(),
			TotalMarkets:               uint64(len(m)),
			TotalMarketsCapitalization: deriveTotalMarketsCapitalization(m),
			Markets:                    m,
			GenerationTime:             uint64(time.Now().Unix()),
			LiquidityMetricsConfig: &markets.LiquidityMetricsConfig{
				MillietherTranches: func() []uint64 {
					tranches := []uint64{}
					for _, tranche := range liquidity.Tranches {
						tranches = append(tranches, tranche.Uint64())
					}
					return tranches
				}(),
			},
		}

		go DebugMarkets(marketsData, m)

		blocker := sync.WaitGroup{}

		blocker.Add(1)
		go func() {
			defer blocker.Done()
			if err := w.Writer.WriteMarketsSummary(summary); err != nil {
				logrus.WithError(err).Errorf("Failed to write markets summary to GCloud storage")
				return
			}
			logrus.WithField("block", header.Number.String()).Infof("Successfully uploaded markets summary")
		}()

		// Write snapshot async since it is not mission critical
		blocker.Add(1)
		go func() {
			defer blocker.Done()
			snapshot := &markets.MarketsSnapshot{
				MarketsSummary: summary,
				MarketInfos:    mapMarketInfos(marketsData),
			}
			if err := w.Writer.WriteMarketsSnapshot(snapshot); err != nil {
				logrus.WithError(err).Errorf("Failed to write markets snapshot to GCloud storage")
				return
			}
			logrus.WithField("block", header.Number.String()).Infof("Successfully uploaded markets snapshot")
		}()

		blocker.Add(1)
		go func() {
			defer blocker.Done()
			details := constructMarketDetails(m, marketsData)
			wg := sync.WaitGroup{}
			for file, _ := range details {
				object, detail := file, details[file]
				wg.Add(1)
				go func() {
					defer wg.Done()
					if err := w.Writer.WriteMarketDetail(object, detail); err != nil {
						logrus.WithError(err).Errorf("Failed to write market detail to GCloud storage")
					}
				}()
			}
			wg.Wait()
			logrus.WithField("block", header.Number.String()).Infof("Successfully uploaded market detail objects")
		}()

		blocker.Wait()
		logrus.WithField("block", header.Number.String()).Infof("Finished processing block")
		lastProcessedBlockNumber = header.Number
	}
}

func constructMarketDetails(ms []*markets.Market, msd *MarketsData) map[string]*markets.MarketDetailByMarketId {
	details := map[string]*markets.MarketDetailByMarketId{}
	for _, market := range ms {
		detail := &markets.MarketDetail{
			MarketId:      market.Id,
			MarketSummary: market,
		}
		if md, ok := msd.ByMarketID[market.Id]; ok {
			detail.MarketInfo = mapMarketInfo(md.Info)
		}
		filename := market.MarketDataSources.MarketDetailFileName
		if _, ok := details[filename]; !ok {
			details[filename] = &markets.MarketDetailByMarketId{
				MarketDetailByMarketId: map[string]*markets.MarketDetail{},
			}
		}
		details[filename].MarketDetailByMarketId[market.Id] = detail
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

func (w *Watcher) translateMarketInfoToMarket(md *MarketData, ethusd, btceth float64) (*markets.Market, error) {
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

	bidsByOutcome, err := GetBids(md.Orders)
	if err != nil {
		logrus.WithError(err).
			WithField("marketInfo", *md.Info).
			Errorf("Failed to get bids by outcome")
		return nil, err
	}
	bestBids := map[uint64]*markets.LiquidityAtPrice{}
	for outcome, list := range bidsByOutcome {
		if len(list.LiquidityAtPrice) <= 0 {
			continue
		}
		bestBids[outcome] = list.LiquidityAtPrice[0]
	}

	asksByOutcome, err := GetAsks(md.Orders)
	if err != nil {
		logrus.WithError(err).
			WithField("marketInfo", *md.Info).
			Errorf("Failed to get asks by outcome")
		return nil, err
	}
	bestAsks := map[uint64]*markets.LiquidityAtPrice{}
	for outcome, asks := range asksByOutcome {
		if len(asks.LiquidityAtPrice) <= 0 {
			continue
		}
		bestAsks[outcome] = asks.LiquidityAtPrice[0]
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

	volume, err := getMarketVolume(md.Info, ethusd, btceth)
	if err != nil {
		logrus.WithError(err).
			WithField("marketInfo", *md.Info).
			Errorf("Failed to get market volume")
		return nil, err
	}

	marketDataSources, err := getMarketDataSources(md.Info.Id)
	if err != nil {
		logrus.WithError(err).
			WithField("marketInfo", *md.Info).
			Errorf("Failed to get market data sources")
		return nil, err
	}

	minPrice, err := strconv.ParseFloat(md.Info.MinPrice, 64)
	if err != nil {
		logrus.WithError(err).
			WithField("marketInfo", *md.Info).
			Errorf("Failed to parse market min price")
		return nil, err
	}
	maxPrice, err := strconv.ParseFloat(md.Info.MaxPrice, 64)
	if err != nil {
		logrus.WithError(err).
			WithField("marketInfo", *md.Info).
			Errorf("Failed to parse market max price")
		return nil, err
	}

	liquidityMetrics := &markets.LiquidityMetrics{
		RetentionRatioByMillietherTranche: map[uint64]float32{},
	}
	// Construct the order books for liquidity calculations
	books := getOutcomeOrderBooks(md.Info, bidsByOutcome, asksByOutcome)
	for _, tranche := range liquidity.Tranches {
		clones := []liquidity.OutcomeOrderBook{}
		for _, book := range books {
			clones = append(clones, book.DeepClone())
		}
		// Determine shares per complete set
		const sellingIncrement = 0.01

		// Ensure the allowance is in the correct denomination
		allowance := tranche.Ether()

		rr := w.LiquidityCalculator.GetLiquidityRetentionRatio(
			sellingIncrement,
			allowance,
			liquidity.MarketData{
				MinPrice: minPrice,
				MaxPrice: maxPrice,
			},
			clones,
		)
		liquidityMetrics.RetentionRatioByMillietherTranche[tranche.Uint64()] = float32(rr)
	}

	_, featured := featuredlist[md.Info.Id]

	// Construct market data
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
		Tags:                 md.Info.Tags,
		IsFeatured:           featured,
		Category:             md.Info.Category,
		LastTradeTime:        md.Info.LastTradeTime,
		BestBids:             bestBids,
		BestAsks:             bestAsks,
		Volume:               volume,
		Bids:                 bidsByOutcome,
		Asks:                 asksByOutcome,
		LiquidityMetrics:     liquidityMetrics,
		MarketDataSources:    marketDataSources,
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
		Usd: float32(outstandingShares * ethusd),
		Btc: float32(outstandingShares / btceth),
	}, nil
}

func getMarketDataSources(id string) (*markets.MarketDataSources, error) {
	// Convert market ID hex to base 10 string
	base10 := fmt.Sprintf("%x", id)
	if len(base10) < 6 {
		return nil, fmt.Errorf("Market ID provided to `getMarketDataSources` is invalid: %s", id)
	}

	// Use the last 6 digits of the ID to bucketize the market
	base10Suffix := base10[(len(base10) - 6):]
	int, err := strconv.Atoi(base10Suffix)
	if err != nil {
		return nil, err
	}

	return &markets.MarketDataSources{
		MarketDetailFileName: strconv.Itoa(int % 10),
	}, nil
}

func getOutcomeOrderBooks(info *augur.MarketInfo, bids, asks map[uint64]*markets.ListLiquidityAtPrice) []liquidity.OutcomeOrderBook {
	books := []liquidity.OutcomeOrderBook{}

	// Helper
	getBidAskLists := func(outcomeID uint64, bidsByOutcome, asksByOutcome map[uint64]*markets.ListLiquidityAtPrice) ([]*markets.LiquidityAtPrice, []*markets.LiquidityAtPrice) {
		bidsList, ok := bidsByOutcome[outcomeID]
		if !ok {
			bidsList = &markets.ListLiquidityAtPrice{
				LiquidityAtPrice: []*markets.LiquidityAtPrice{},
			}
		}
		asksList, ok := asksByOutcome[outcomeID]
		if !ok {
			asksList = &markets.ListLiquidityAtPrice{
				LiquidityAtPrice: []*markets.LiquidityAtPrice{},
			}
		}
		return bidsList.LiquidityAtPrice, asksList.LiquidityAtPrice
	}

	// Construct the OutcomeOrderBook based on market type
	switch strings.ToLower(info.MarketType) {
	case "yesno", "scalar":
		for _, outcome := range info.Outcomes {
			// Create only the OutcomeOrderBook for the yes & upper outcomes
			if outcome.Id == 1 {
				book := liquidity.NewOutcomeOrderBook(getBidAskLists(outcome.Id, bids, asks))
				books = append(books, book)
			}
		}
	default: // "categorical"
		for _, outcome := range info.Outcomes {
			book := liquidity.NewOutcomeOrderBook(getBidAskLists(outcome.Id, bids, asks))
			books = append(books, book)
		}
	}
	return books
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

func getMarketVolume(info *augur.MarketInfo, ethusd, btceth float64) (*markets.Price, error) {
	volume, err := strconv.ParseFloat(info.Volume, 64)
	if err != nil {
		return nil, err
	}
	return &markets.Price{
		Eth: float32(volume),
		Usd: float32(volume * ethusd),
		Btc: float32(volume / btceth),
	}, nil
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
		LastTradeTime:        info.LastTradeTime,
		LastTradeBlockNumber: info.LastTradeBlockNumber,
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
