package markets

import (
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/stateshape/augur-analyzer/pkg/augur"
)

func PrintMarketInfoByDescription(infoByAddress map[string]augur.MarketInfo, description string) {
	for _, info := range infoByAddress {
		if strings.Contains(info.Description, description) {
			state, _ := augur.ReportingState_name[int32(info.ReportingState)]
			logrus.WithFields(logrus.Fields{
				"marketId":         info.Id,
				"minPrice":         info.MinPrice,
				"maxPrice":         info.MaxPrice,
				"oustandingShares": info.OutstandingShares,
				"reportingState":   state,
				"description":      info.Description,
				"details":          info.Details,
				"tickSize":         info.TickSize,
			})
		}
	}
}

func PrintMarketInfoByAddress(infoByAddress map[string]augur.MarketInfo, address string) {
	info, ok := infoByAddress[address]
	if !ok {
		logrus.Infof("Market with address %s not found", address)
		return
	}
	state, _ := augur.ReportingState_name[int32(info.ReportingState)]
	logrus.WithFields(logrus.Fields{
		"marketId":         info.Id,
		"minPrice":         info.MinPrice,
		"maxPrice":         info.MaxPrice,
		"oustandingShares": info.OutstandingShares,
		"reportingState":   state,
		"description":      info.Description,
		"details":          info.Details,
		"tickSize":         info.TickSize,
	})
}
