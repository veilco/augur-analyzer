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

func getYesNoPredictions(m *Market, outcomes []*Outcome) []*markets.Prediction {
	no := outcomes[0]
	yes := outcomes[1]

	// If the market has no volume, do not return predictions
	if no.Volume <= 0.0 && yes.Volume <= 0.0 {
		return []*markets.Prediction{}
	}

	// Derive predicted percent
	percent := yes.Price
	if yes.Volume <= 0.0 && no.Volume > 0.0 {
		percent = 1.0 - no.Price
	}

	return []*markets.Prediction{
		{
			Name:    "yes",
			Percent: float32(percent * 100),
		},
	}
}

func getCategoricalPredictions(m *Market, outcomes []*Outcome) []*markets.Prediction {
	// If the market has no volume, do not return predictions
	hasVolume := false
	for _, o := range outcomes {
		if o.Volume > 0.0 {
			hasVolume = true
		}
	}
	if !hasVolume {
		return []*markets.Prediction{}
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
		})
	}
	return predictions
}

func getScalarPredictions(m *Market, outcomes []*Outcome) []*markets.Prediction {
	if len(outcomes) != 2 {
		logrus.WithField("outcomeInfos", outcomes).Errorf("`getScalarPredictions` was called without 2 `OutcomeInfo` arguments")
		return []*markets.Prediction{}
	}
	lower := outcomes[0]
	upper := outcomes[1]

	// If the market has no volume, do not return predictions
	if lower.Volume == 0.0 && upper.Volume == 0.0 {
		return []*markets.Prediction{}
	}

	// Derive predicted scalar value
	value := upper.Price
	if upper.Volume <= 0.0 && lower.Volume > 0.0 {
		value = m.MaxPrice + m.MinPrice - lower.Price
	}

	return []*markets.Prediction{
		{
			Name:  "",
			Value: float32(value),
		},
	}
}
