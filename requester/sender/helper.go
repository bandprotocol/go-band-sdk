package sender

import (
	"fmt"
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
)

// GetRequestID returns the first request id from event of transaction
func GetRequestID(events []sdk.StringEvent) (uint64, error) {
	for _, event := range events {
		if event.Type == "request" {
			rid, err := strconv.ParseUint(event.Attributes[0].Value, 10, 64)
			if err != nil {
				return 0, err
			}

			return rid, nil
		}
	}
	return 0, fmt.Errorf("cannot find request id")
}

// GetDsFee returns the total fee of the first request from event of transaction
func GetDsFee(events []sdk.StringEvent) (uint64, error) {
	for _, event := range events {
		if event.Type == "request" {
			for _, attribute := range event.Attributes {
				if attribute.Key == "total_fees" {
					if attribute.Value == "" {
						return 0, nil
					}
					// TotalFees is mean all ds fees
					totalFees, err := sdk.ParseCoinsNormalized(attribute.Value)
					if err != nil {
						return 0, err
					}

					return totalFees.AmountOf("uband").Uint64(), nil
				}
			}
		}
	}
	return 0, fmt.Errorf("cannot find total fees")
}

func decodeTxFromRes(res *sdk.TxResponse) (height int64, timestamp int64, txFee uint64, err error) {
	resTimestamp, err := time.Parse(time.RFC3339, res.Timestamp)
	if err != nil {
		return 0, 0, 0, err
	}

	txDecode, ok := res.Tx.GetCachedValue().(*txtypes.Tx)
	if !ok {
		return 0, 0, 0, err
	}

	return res.Height, resTimestamp.Unix(), txDecode.AuthInfo.Fee.Amount.AmountOf("uband").Uint64(), nil
}
