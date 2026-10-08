package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/evm/encoding"
	legacytypes "github.com/cosmos/evm/rpc/types/legacy"
	"github.com/cosmos/evm/testutil/constants"

	sdkmath "cosmossdk.io/math"
)

// Pre-v9 blocks store Ethereum txs as ethermint.evm.v1 messages. Decoding one
// with a non-empty access list resolves ethermint.evm.v1.AccessTuple by name.
func TestDecodeLegacyTxWithAccessList(t *testing.T) {
	evmChainID := constants.ExampleChainID.EVMChainID
	chainID := sdkmath.NewIntFromUint64(evmChainID)
	zero := sdkmath.ZeroInt()
	accessList := legacytypes.AccessList{{
		Address:     "0x0000000000000000000000000000000000000001",
		StorageKeys: []string{"0x0000000000000000000000000000000000000000000000000000000000000000"},
	}}

	testCases := []struct {
		name   string
		txData legacytypes.TxData
	}{
		{
			name: "access list tx",
			txData: &legacytypes.AccessListTx{
				ChainID:  &chainID,
				Amount:   &zero,
				GasPrice: &zero,
				Accesses: accessList,
			},
		},
		{
			name: "dynamic fee tx",
			txData: &legacytypes.DynamicFeeTx{
				ChainID:   &chainID,
				Amount:    &zero,
				GasTipCap: &zero,
				GasFeeCap: &zero,
				Accesses:  accessList,
			},
		},
	}

	encodingConfig := encoding.MakeConfig(evmChainID)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := legacytypes.PackTxData(tc.txData)
			require.NoError(t, err)

			txBuilder := encodingConfig.TxConfig.NewTxBuilder()
			err = txBuilder.SetMsgs(&legacytypes.MsgEthereumTx{Data: data})
			require.NoError(t, err)

			bz, err := encodingConfig.TxConfig.TxEncoder()(txBuilder.GetTx())
			require.NoError(t, err)

			tx, err := encodingConfig.TxConfig.TxDecoder()(bz)
			require.NoError(t, err)

			msgs := tx.GetMsgs()
			require.Len(t, msgs, 1)
			msg, ok := msgs[0].(*legacytypes.MsgEthereumTx)
			require.True(t, ok)
			txData, err := legacytypes.UnpackTxData(msg.Data)
			require.NoError(t, err)
			require.Equal(t, *accessList.ToEthAccessList(), txData.GetAccessList())
		})
	}
}
