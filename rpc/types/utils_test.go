package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"
)

// stateDBCommitFailureLog is the verbatim `log` field CometBFT returned for tx 0 of
// XRPL EVM testnet block 8005874, an EVM tx that sent a non-zero native value to the
// ecrecover precompile (ethm1qqq...p4m6ged is the bech32 form of 0x00..01, a blocked
// address). The bank module rejected the balance write, so the whole cosmos tx failed
// with code 4 and carried no response payload.
//
// Such a tx used to satisfy TxSucessOrExpectedFailure, so it was included in the
// ethereum view of the block; the receipt builder then tried to decode its logs from
// the empty payload and returned "invalid message index: 0", which is fatal for the
// whole block and permanently broke every block-level RPC at that height.
const stateDBCommitFailureLog = "failed to execute message; message index: 0: " +
	"failed to apply transaction: failed to apply ethereum core message: " +
	"failed to commit stateDB: failed to set account: " +
	"ethm1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqp4m6ged is not allowed to receive funds: unauthorized"

func TestTxSucessOrExpectedFailure(t *testing.T) {
	testCases := []struct {
		name     string
		res      *abci.ExecTxResult
		expected bool
	}{
		{
			// Regression: xrplevm testnet block 8005874. Must stay excluded.
			name:     "stateDB commit failure is excluded",
			res:      &abci.ExecTxResult{Code: 4, Log: stateDBCommitFailureLog},
			expected: false,
		},
		{
			name:     "successful tx is included",
			res:      &abci.ExecTxResult{Code: 0},
			expected: true,
		},
		{
			// The fee is deducted in the ante handler, so this class must stay
			// included even though it also fails with a non-zero code.
			name: "tx exceeding the block gas limit is still included",
			res: &abci.ExecTxResult{
				Code: 11,
				Log:  "out of gas in location: block gas meter; gasWanted: 126044",
			},
			expected: true,
		},
		{
			name:     "unrelated failure is excluded",
			res:      &abci.ExecTxResult{Code: 5, Log: "insufficient funds"},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, TxSucessOrExpectedFailure(tc.res))
		})
	}
}
