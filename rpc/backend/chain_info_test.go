package backend

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	ethrpc "github.com/ethereum/go-ethereum/rpc"
	"google.golang.org/grpc/metadata"

	"github.com/cometbft/cometbft/abci/types"
	tmrpctypes "github.com/cometbft/cometbft/rpc/core/types"

	"github.com/cosmos/evm/rpc/backend/mocks"
	rpc "github.com/cosmos/evm/rpc/types"
	utiltx "github.com/cosmos/evm/testutil/tx"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *BackendTestSuite) TestBaseFee() {
	baseFee := math.NewInt(1)

	testCases := []struct {
		name         string
		blockRes     *tmrpctypes.ResultBlockResults
		registerMock func()
		expBaseFee   *big.Int
		expPass      bool
	}{
		{
			"fail - grpc BaseFee error",
			&tmrpctypes.ResultBlockResults{Height: 1},
			func() {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFeeError(queryClient)
			},
			nil,
			false,
		},
		{
			"fail - grpc BaseFee error - with non feemarket block event",
			&tmrpctypes.ResultBlockResults{
				Height: 1,
				FinalizeBlockEvents: []types.Event{
					{
						Type: evmtypes.EventTypeBlockBloom,
					},
				},
			},
			func() {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFeeError(queryClient)
			},
			nil,
			false,
		},
		{
			"fail - grpc BaseFee error - with feemarket block event",
			&tmrpctypes.ResultBlockResults{
				Height: 1,
				FinalizeBlockEvents: []types.Event{
					{
						Type: evmtypes.EventTypeFeeMarket,
					},
				},
			},
			func() {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFeeError(queryClient)
			},
			nil,
			false,
		},
		{
			"fail - grpc BaseFee error - with feemarket block event with wrong attribute value",
			&tmrpctypes.ResultBlockResults{
				Height: 1,
				FinalizeBlockEvents: []types.Event{
					{
						Type: evmtypes.EventTypeFeeMarket,
						Attributes: []types.EventAttribute{
							{Value: "/1"},
						},
					},
				},
			},
			func() {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFeeError(queryClient)
			},
			nil,
			false,
		},
		{
			"fail - grpc baseFee error - with feemarket block event with baseFee attribute value",
			&tmrpctypes.ResultBlockResults{
				Height: 1,
				FinalizeBlockEvents: []types.Event{
					{
						Type: evmtypes.EventTypeFeeMarket,
						Attributes: []types.EventAttribute{
							{Value: baseFee.String()},
						},
					},
				},
			},
			func() {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFeeError(queryClient)
			},
			baseFee.BigInt(),
			true,
		},
		{
			"fail - base fee or london fork not enabled",
			&tmrpctypes.ResultBlockResults{Height: 1},
			func() {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFeeDisabled(queryClient)
			},
			nil,
			true,
		},
		{
			"pass",
			&tmrpctypes.ResultBlockResults{Height: 1},
			func() {
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterBaseFee(queryClient, baseFee)
			},
			baseFee.BigInt(),
			true,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock()

			baseFee, err := suite.backend.BaseFee(tc.blockRes)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(tc.expBaseFee, baseFee)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestChainId() {
	expChainID := (*hexutil.Big)(big.NewInt(262144))
	testCases := []struct {
		name         string
		registerMock func()
		expChainID   *hexutil.Big
		expPass      bool
	}{
		{
			"pass - block is at or past the EIP-155 replay-protection fork block, return chainID from config ",
			func() {
				var header metadata.MD
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterParamsInvalidHeight(queryClient, &header, int64(1))
			},
			expChainID,
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("case %s", tc.name), func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock()

			chainID, err := suite.backend.ChainID()
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(tc.expChainID, chainID)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestGetCoinbase() {
	validatorAcc := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	testCases := []struct {
		name         string
		registerMock func()
		accAddr      sdk.AccAddress
		expPass      bool
	}{
		{
			"fail - Can't retrieve status from node",
			func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				RegisterStatusError(client)
			},
			validatorAcc,
			false,
		},
		{
			"fail - Can't query validator account",
			func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterStatus(client)
				RegisterValidatorAccountError(queryClient)
			},
			validatorAcc,
			false,
		},
		{
			"pass - Gets coinbase account",
			func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterStatus(client)
				RegisterValidatorAccount(queryClient, validatorAcc)
			},
			validatorAcc,
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("case %s", tc.name), func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock()

			accAddr, err := suite.backend.GetCoinbase()

			if tc.expPass {
				suite.Require().Equal(tc.accAddr, accAddr)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestSuggestGasTipCap() {
	testCases := []struct {
		name         string
		registerMock func()
		baseFee      *big.Int
		expGasTipCap *big.Int
		expPass      bool
	}{
		{
			"pass - London hardfork not enabled or feemarket not enabled ",
			func() {},
			nil,
			big.NewInt(0),
			true,
		},
		{
			"pass - Gets the suggest gas tip cap ",
			func() {},
			nil,
			big.NewInt(0),
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("case %s", tc.name), func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock()

			maxDelta, err := suite.backend.SuggestGasTipCap(tc.baseFee)

			if tc.expPass {
				suite.Require().Equal(tc.expGasTipCap, maxDelta)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestGlobalMinGasPrice() {
	testCases := []struct {
		name           string
		registerMock   func()
		expMinGasPrice *big.Int
		expPass        bool
	}{
		{
			"pass - get GlobalMinGasPrice",
			func() {
				qc := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterGlobalMinGasPrice(qc, 1)
			},
			big.NewInt(1),
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("case %s", tc.name), func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock()

			globalMinGasPrice, err := suite.backend.GlobalMinGasPrice()

			if tc.expPass {
				suite.Require().Equal(tc.expMinGasPrice, globalMinGasPrice)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestFeeHistory() {
	testCases := []struct {
		name           string
		registerMock   func(validator sdk.AccAddress)
		userBlockCount ethrpc.BlockNumber
		latestBlock    ethrpc.BlockNumber
		expFeeHistory  *rpc.FeeHistoryResult
		validator      sdk.AccAddress
		expPass        bool
	}{
		{
			"fail - can't get params ",
			func(_ sdk.AccAddress) {
				var header metadata.MD
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				suite.backend.cfg.JSONRPC.FeeHistoryCap = 0
				RegisterParamsError(queryClient, &header, ethrpc.BlockNumber(1).Int64())
			},
			1,
			-1,
			nil,
			nil,
			false,
		},
		{
			"fail - user block count higher than max block count ",
			func(_ sdk.AccAddress) {
				var header metadata.MD
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				suite.backend.cfg.JSONRPC.FeeHistoryCap = 0
				RegisterParams(queryClient, &header, ethrpc.BlockNumber(1).Int64())
			},
			1,
			-1,
			nil,
			nil,
			false,
		},
		{
			"fail - Tendermint block fetching error ",
			func(_ sdk.AccAddress) {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				suite.backend.cfg.JSONRPC.FeeHistoryCap = 2
				RegisterBlockError(client, ethrpc.BlockNumber(1).Int64())
			},
			1,
			1,
			nil,
			nil,
			false,
		},
		{
			"fail - Eth block fetching error",
			func(sdk.AccAddress) {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				suite.backend.cfg.JSONRPC.FeeHistoryCap = 2
				_, err := RegisterBlock(client, ethrpc.BlockNumber(1).Int64(), nil)
				suite.Require().NoError(err)
				RegisterBlockResultsError(client, 1)
			},
			1,
			1,
			nil,
			nil,
			true,
		},
		{
			"fail - Invalid base fee",
			func(validator sdk.AccAddress) {
				// baseFee := math.NewInt(1)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				suite.backend.cfg.JSONRPC.FeeHistoryCap = 2
				_, err := RegisterBlock(client, ethrpc.BlockNumber(1).Int64(), nil)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFeeError(queryClient)
				RegisterValidatorAccount(queryClient, validator)
				RegisterConsensusParams(client, 1)
			},
			1,
			1,
			nil,
			sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
			false,
		},
		{
			"pass - Valid FeeHistoryResults object",
			func(validator sdk.AccAddress) {
				var header metadata.MD
				baseFee := math.NewInt(1)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				suite.backend.cfg.JSONRPC.FeeHistoryCap = 2
				_, err := RegisterBlock(client, ethrpc.BlockNumber(1).Int64(), nil)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFee(queryClient, baseFee)
				RegisterValidatorAccount(queryClient, validator)
				RegisterConsensusParams(client, 1)
				RegisterParams(queryClient, &header, 1)
			},
			1,
			1,
			&rpc.FeeHistoryResult{
				OldestBlock:  (*hexutil.Big)(big.NewInt(1)),
				BaseFee:      []*hexutil.Big{(*hexutil.Big)(big.NewInt(1)), (*hexutil.Big)(big.NewInt(1))},
				GasUsedRatio: []float64{0},
				Reward:       [][]*hexutil.Big{{(*hexutil.Big)(big.NewInt(0)), (*hexutil.Big)(big.NewInt(0)), (*hexutil.Big)(big.NewInt(0)), (*hexutil.Big)(big.NewInt(0))}},
			},
			sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("case %s", tc.name), func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock(tc.validator)

			feeHistory, err := suite.backend.FeeHistory(tc.userBlockCount, tc.latestBlock, []float64{25, 50, 75, 100})
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(feeHistory, tc.expFeeHistory)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
