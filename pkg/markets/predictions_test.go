package markets_test

import (
	"testing"

	"github.com/stateshape/augur-analyzer/pkg/markets"
	"github.com/stateshape/augur-analyzer/pkg/proto/augur"
	protomarkets "github.com/stateshape/augur-analyzer/pkg/proto/markets"

	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGetBids(t *testing.T) {
	cases := []struct {
		Name         string
		Orders       *augur.GetOrdersResponse_OrdersByOrderIdByOrderTypeByOutcome
		ExpectedBids map[uint64]*protomarkets.ListLiquidityAtPrice
	}{
		{
			Name: "1",
			Orders: &augur.GetOrdersResponse_OrdersByOrderIdByOrderTypeByOutcome{
				OrdersByOrderIdByOrderTypeByOutcome: map[uint64]*augur.GetOrdersResponse_OrdersByOrderIdByOrderType{
					1: &augur.GetOrdersResponse_OrdersByOrderIdByOrderType{
						BuyOrdersByOrderId: &augur.GetOrdersResponse_OrdersByOrderId{
							OrdersByOrderId: map[string]*augur.Order{
								uuid.New(): &augur.Order{
									OrderId:         uuid.New(),
									TransactionHash: uuid.New(),
									Price:           ".9",
									Amount:          "10",
									OrderState:      augur.OrderState_OPEN,
								},
								uuid.New(): &augur.Order{
									Price:      ".9",
									Amount:     "10",
									OrderState: augur.OrderState_OPEN,
								},
							},
						},
						SellOrdersByOrderId: &augur.GetOrdersResponse_OrdersByOrderId{
							OrdersByOrderId: map[string]*augur.Order{},
						},
					},
				},
			},
			ExpectedBids: map[uint64]*protomarkets.ListLiquidityAtPrice{
				1: &protomarkets.ListLiquidityAtPrice{
					LiquidityAtPrice: []*protomarkets.LiquidityAtPrice{
						{
							Price:  .9,
							Amount: 20,
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			bids, err := markets.GetBids(c.Orders)
			assert.Nil(t, err)
			assert.Equal(t, len(c.ExpectedBids), len(bids))
			for outcomeID, ll := range c.ExpectedBids {
				listliquidity, ok := bids[outcomeID]
				assert.True(t, ok)
				assert.Equal(t, len(listliquidity.LiquidityAtPrice), len(ll.LiquidityAtPrice))
				for i := 0; i < len(ll.LiquidityAtPrice); i++ {
					assert.Equal(t, listliquidity.LiquidityAtPrice[i].Price, ll.LiquidityAtPrice[i].Price)
					assert.Equal(t, listliquidity.LiquidityAtPrice[i].Amount, ll.LiquidityAtPrice[i].Amount)
				}
			}
		})
	}
}

func TestGetAsks(t *testing.T) {

}
