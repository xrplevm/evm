package ante_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/evm/ante"
	ethante "github.com/cosmos/evm/ante/evm"
	evmdante "github.com/cosmos/evm/evmd/ante"
	"github.com/cosmos/evm/evmd/tests/integration"
	"github.com/cosmos/evm/testutil/integration/evm/network"
	"github.com/cosmos/evm/types"
)

//nolint:thelper // RunValidateHandlerOptionsTest is not a helper function; it's an externally called benchmark entry point
func RunValidateHandlerOptionsTest(t *testing.T, create network.CreateEvmApp, options ...network.ConfigOption) {
	nw := network.NewUnitTestNetwork(create, options...)
	cases := []struct {
		name    string
		options evmdante.HandlerOptions
		expPass bool
	}{
		{
			"fail - empty options",
			evmdante.HandlerOptions{},
			false,
		},
		{
			"fail - empty account keeper",
			evmdante.HandlerOptions{
				Cdc:           nw.App.AppCodec(),
				AccountKeeper: nil,
			},
			false,
		},
		{
			"fail - empty bank keeper",
			evmdante.HandlerOptions{
				Cdc:           nw.App.AppCodec(),
				AccountKeeper: nw.App.GetAccountKeeper(),
				BankKeeper:    nil,
			},
			false,
		},
		{
			"fail - empty IBC keeper",
			evmdante.HandlerOptions{
				Cdc:           nw.App.AppCodec(),
				AccountKeeper: nw.App.GetAccountKeeper(),
				BankKeeper:    nw.App.GetBankKeeper(),
				IBCKeeper:     nil,
			},
			false,
		},
		{
			"fail - empty fee market keeper",
			evmdante.HandlerOptions{
				Cdc:             nw.App.AppCodec(),
				AccountKeeper:   nw.App.GetAccountKeeper(),
				BankKeeper:      nw.App.GetBankKeeper(),
				IBCKeeper:       nw.App.GetIBCKeeper(),
				FeeMarketKeeper: nil,
			},
			false,
		},
		{
			"fail - empty EVM keeper",
			evmdante.HandlerOptions{
				Cdc:             nw.App.AppCodec(),
				AccountKeeper:   nw.App.GetAccountKeeper(),
				BankKeeper:      nw.App.GetBankKeeper(),
				IBCKeeper:       nw.App.GetIBCKeeper(),
				FeeMarketKeeper: nw.App.GetFeeMarketKeeper(),
				EvmKeeper:       nil,
			},
			false,
		},
		{
			"fail - empty signature gas consumer",
			evmdante.HandlerOptions{
				Cdc:             nw.App.AppCodec(),
				AccountKeeper:   nw.App.GetAccountKeeper(),
				BankKeeper:      nw.App.GetBankKeeper(),
				IBCKeeper:       nw.App.GetIBCKeeper(),
				FeeMarketKeeper: nw.App.GetFeeMarketKeeper(),
				EvmKeeper:       nw.App.GetEVMKeeper(),
				SigGasConsumer:  nil,
			},
			false,
		},
		{
			"fail - empty signature mode handler",
			evmdante.HandlerOptions{
				Cdc:             nw.App.AppCodec(),
				AccountKeeper:   nw.App.GetAccountKeeper(),
				BankKeeper:      nw.App.GetBankKeeper(),
				IBCKeeper:       nw.App.GetIBCKeeper(),
				FeeMarketKeeper: nw.App.GetFeeMarketKeeper(),
				EvmKeeper:       nw.App.GetEVMKeeper(),
				SigGasConsumer:  ante.SigVerificationGasConsumer,
				SignModeHandler: nil,
			},
			false,
		},
		{
			"fail - empty tx fee checker",
			evmdante.HandlerOptions{
				Cdc:             nw.App.AppCodec(),
				AccountKeeper:   nw.App.GetAccountKeeper(),
				BankKeeper:      nw.App.GetBankKeeper(),
				IBCKeeper:       nw.App.GetIBCKeeper(),
				FeeMarketKeeper: nw.App.GetFeeMarketKeeper(),
				EvmKeeper:       nw.App.GetEVMKeeper(),
				SigGasConsumer:  ante.SigVerificationGasConsumer,
				SignModeHandler: nw.App.GetTxConfig().SignModeHandler(),
				TxFeeChecker:    nil,
			},
			false,
		},
		{
			"success - default app options",
			evmdante.HandlerOptions{
				Cdc:                    nw.App.AppCodec(),
				AccountKeeper:          nw.App.GetAccountKeeper(),
				BankKeeper:             nw.App.GetBankKeeper(),
				ExtensionOptionChecker: types.HasDynamicFeeExtensionOption,
				EvmKeeper:              nw.App.GetEVMKeeper(),
				FeegrantKeeper:         nw.App.GetFeeGrantKeeper(),
				IBCKeeper:              nw.App.GetIBCKeeper(),
				FeeMarketKeeper:        nw.App.GetFeeMarketKeeper(),
				SignModeHandler:        nw.GetEncodingConfig().TxConfig.SignModeHandler(),
				SigGasConsumer:         ante.SigVerificationGasConsumer,
				MaxTxGasWanted:         40000000,
				TxFeeChecker:           ethante.NewDynamicFeeChecker(nw.App.GetFeeMarketKeeper()),
			},
			true,
		},
	}

	for _, tc := range cases {
		err := tc.options.Validate()
		if tc.expPass {
			require.NoError(t, err, tc.name)
		} else {
			require.Error(t, err, tc.name)
		}
	}
}

func TestValidateHandlerOptions(t *testing.T) {
	RunValidateHandlerOptionsTest(t, integration.CreateEvmd)
}
