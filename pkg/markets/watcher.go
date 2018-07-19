package markets

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/stateshape/augur-analyzer/pkg/augur"
	"github.com/stateshape/augur-analyzer/pkg/env"
	"github.com/stateshape/augur-analyzer/pkg/gcloud"
	"github.com/stateshape/augur-analyzer/pkg/pricing"
	"github.com/stateshape/augur-analyzer/pkg/proto/markets"

	"cloud.google.com/go/storage"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/golang/protobuf/proto"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	MARKETS_SUMMARIES_OBJECT = "markets.pb"
	MARKETS_SNAPSHOT_OBJECT  = "snapshot"
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
	StorageAPI *storage.Client
	Cache      *WatcherCache
}

// TODO: make this structure safe for concurrent operations
type WatcherCache struct {
	LastMarketsSummary []byte
}

type BlockHeaderSubscriber struct {
	Subscription      ethereum.Subscription
	BlockHeaderStream chan *types.Header
}

func NewWatcher(pricingAPI pricing.PricingClient, web3API *ethclient.Client, augurAPI augur.MarketsApiClient, storageAPI *storage.Client) *Watcher {
	return &Watcher{pricingAPI, web3API, augurAPI, storageAPI, &WatcherCache{
		LastMarketsSummary: []byte{},
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
		time.Sleep(time.Second * 15)
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

		// Query market info
		marketAddresses := getMarketsResponse.MarketAddresses
		infosByAddress := map[string]*augur.MarketInfo{}
		for x := 0; x < len(marketAddresses); x += 10 {
			limit := x + 10
			if limit > len(marketAddresses) {
				limit = len(marketAddresses)
			}
			addresses := marketAddresses[x:limit]
			getMarketsInfoResponse, err := w.AugurAPI.GetMarketsInfo(context.TODO(), &augur.GetMarketsInfoRequest{
				MarketAddresses: addresses,
			})
			if err != nil {
				logrus.WithError(err).WithField("block", header.Number.String()).
					Errorf("Call to augur-node `GetMarketsInfo` failed")
				continue
			}
			for _, mi := range getMarketsInfoResponse.MarketInfo {
				infosByAddress[mi.Id] = mi
			}
		}

		// Construct market summary structure and serialize to Protobuf
		ethusd, err := w.PricingAPI.ETHtoUSD()
		if err != nil {
			logrus.WithError(err).Errorf("Failed to get ETH USD exchange rate")
			continue
		}
		btceth, err := w.PricingAPI.BTCtoETH()
		if err != nil {
			logrus.WithError(err).Errorf("Failed to get BTC ETH exchange rate")
			continue
		}

		m := []*markets.Market{}
		for _, info := range infosByAddress {
			if _, ok := blacklist[info.Id]; ok {
				logrus.WithFields(logrus.Fields{
					"id":          info.Id,
					"description": info.Description,
				}).Infof("Skipping blacklisted market")
				continue
			}

			market, err := translateMarketInfoToMarket(info, ethusd, btceth)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"block":         header.Number.String(),
					"marketAddress": info.Id,
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

		snapshot := &markets.MarketsSnapshot{
			MarketsSummary: summary,
			MarketInfos:    mapMarketInfos(infosByAddress),
		}

		serializedSummary, err := proto.Marshal(summary)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"blockNumber": header.Number.String(),
			}).WithError(err).Errorf("Failed to protobuf serialize market summary")
			continue
		}

		w.Cache.LastMarketsSummary = serializedSummary

		logrus.Infof("Successfully serialized a market summary for block #%s", header.Number.String())

		// Upload to Google Cloud
		if err := gcloud.WriteObject(w.StorageAPI, gcloud.WriteObjectParameters{
			Bucket:     viper.GetString(env.GCloudStorageBucket),
			ObjectName: MARKETS_SUMMARIES_OBJECT,
			Content:    serializedSummary,
		}, func(wrtr *storage.Writer) {
			wrtr.ContentType = "application/octet-stream"
			wrtr.CacheControl = "public, max-age=15"
			wrtr.ACL = []storage.ACLRule{
				{storage.AllUsers, storage.RoleReader},
			}
		}); err != nil {
			logrus.WithError(err).Errorf("Failed to write markets summary to GCloud storage")
			continue
		}

		lastProcessedBlockNumber = header.Number

		go func() {
			serializedSnapshot, err := proto.Marshal(snapshot)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"blockNumber": header.Number.String(),
				}).WithError(fmt.Errorf("Failed to protobuf serialize markets snapshot"))
				return
			}

			if err := gcloud.WriteObject(w.StorageAPI, gcloud.WriteObjectParameters{
				Bucket:     viper.GetString(env.GCloudStorageBucket),
				ObjectName: MARKETS_SNAPSHOT_OBJECT,
				Content:    serializedSnapshot,
			}, func(wrtr *storage.Writer) {
				wrtr.ContentType = "application/octet-stream"
				wrtr.CacheControl = "public, max-age=15"
				wrtr.ACL = []storage.ACLRule{
					{storage.AllUsers, storage.RoleReader},
				}
			}); err != nil {
				logrus.WithError(err).Errorf("Failed to write markets snapshot to GCloud storage")
				return
			}
			logrus.WithField("blockNumber", header.Number.String()).Infof("Uploaded snapshot file")
		}()

		logrus.WithField("blockNumber", header.Number.String()).Infof("Finished processing block")
	}
}

func (w *Watcher) getNewBlockHeadersSubscription() (*BlockHeaderSubscriber, error) {
	blockHeaderStream := make(chan *types.Header)
	subscription, err := w.Web3API.SubscribeNewHead(context.TODO(), blockHeaderStream)
	if err != nil {
		return nil, err
	}
	return &BlockHeaderSubscriber{
		Subscription:      subscription,
		BlockHeaderStream: blockHeaderStream,
	}, nil

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

func translateMarketInfoToMarket(info *augur.MarketInfo, ethusd, btceth float64) (*markets.Market, error) {
	if info == nil {
		return nil, fmt.Errorf("`translateMarketInfoToMarket` required a non nil MarketInfo as an argument")
	}

	marketCapitalization, err := translateMarketInfoToMarketCapitalization(info, ethusd, btceth)
	if err != nil {
		logrus.WithError(err).
			WithField("marketInfo", *info).
			Errorf("Failed to translate market info into market capitalization")
		return nil, err
	}
	predictions, err := getPredictions(info)
	if err != nil {
		logrus.WithError(err).
			WithField("marketInfo", *info).
			Errorf("Failed to translate market info into predictions")
		return nil, err
	}
	marketType, err := getMarketType(info)
	if err != nil {
		logrus.WithError(err).
			WithField("marketInfo", *info).
			Errorf("Failed to get market type")
		return nil, err
	}

	_, featured := featuredlist[info.Id]

	return &markets.Market{
		Id:                   info.Id,
		MarketType:           marketType,
		Name:                 info.Description,
		CommentCount:         0,
		MarketCapitalization: marketCapitalization,
		EndDate:              info.EndTime,
		Predictions:          predictions,
		Author:               info.Author,
		CreationTime:         info.CreationTime,
		CreationBlock:        info.CreationBlock,
		ResolutionSource:     info.ResolutionSource,
		Details:              info.Details,
		Tags:                 info.Tags,
		IsFeatured:           featured,
		Category:             info.Category,
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

func getPredictions(info *augur.MarketInfo) ([]*markets.Prediction, error) {
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
		predictions = append(predictions, getYesNoPredictions(m, os)...)
	case MarketTypeCategorical:
		predictions = append(predictions, getCategoricalPredictions(m, os)...)
	case MarketTypeScalar:
		predictions = append(predictions, getScalarPredictions(m, os)...)
	}
	return predictions, nil
}

func mapMarketInfos(infosByAddress map[string]*augur.MarketInfo) []*markets.MarketInfo {
	mis := []*markets.MarketInfo{}
	for id, info := range infosByAddress {
		if _, ok := blacklist[id]; ok {
			continue
		}

		// Map equal types
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
		mis = append(mis, m)
	}
	return mis
}
