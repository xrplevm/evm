package erc20

import (
	"fmt"
	"math/big"

	utiltx "github.com/cosmos/evm/testutil/tx"
	"github.com/cosmos/evm/x/erc20/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

func (suite *PrecompileTestSuite) TestMintingEnabled() {
	var ctx sdk.Context
	sender := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	receiver := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	expPair := types.NewTokenPair(utiltx.GenerateAddress(), "coin", types.OWNER_MODULE)
	id := expPair.GetID()

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"conversion is disabled globally",
			func() {
				params := types.DefaultParams()
				params.EnableErc20 = false
				suite.network.App.GetErc20Keeper().SetParams(ctx, params) //nolint:errcheck
			},
			false,
		},
		{
			"token pair not found",
			func() {},
			false,
		},
		{
			"conversion is disabled for the given pair",
			func() {
				expPair.Enabled = false
				suite.network.App.GetErc20Keeper().SetTokenPair(ctx, expPair)
				suite.network.App.GetErc20Keeper().SetDenomMap(ctx, expPair.Denom, id)
				suite.network.App.GetErc20Keeper().SetERC20Map(ctx, expPair.GetERC20Contract(), id)
			},
			false,
		},
		{
			"token transfers are disabled",
			func() {
				expPair.Enabled = true
				suite.network.App.GetErc20Keeper().SetTokenPair(ctx, expPair)
				suite.network.App.GetErc20Keeper().SetDenomMap(ctx, expPair.Denom, id)
				suite.network.App.GetErc20Keeper().SetERC20Map(ctx, expPair.GetERC20Contract(), id)

				suite.network.App.GetBankKeeper().SetSendEnabled(ctx, expPair.Denom, false)
			},
			false,
		},
		{
			"token not registered",
			func() {
				suite.network.App.GetErc20Keeper().SetDenomMap(ctx, expPair.Denom, id)
				suite.network.App.GetErc20Keeper().SetERC20Map(ctx, expPair.GetERC20Contract(), id)
			},
			false,
		},
		{
			"receiver address is blocked (module account)",
			func() {
				suite.network.App.GetErc20Keeper().SetTokenPair(ctx, expPair)
				suite.network.App.GetErc20Keeper().SetDenomMap(ctx, expPair.Denom, id)
				suite.network.App.GetErc20Keeper().SetERC20Map(ctx, expPair.GetERC20Contract(), id)

				acc := suite.network.App.GetAccountKeeper().GetModuleAccount(ctx, types.ModuleName)
				receiver = acc.GetAddress()
			},
			false,
		},
		{
			"ok",
			func() {
				suite.network.App.GetErc20Keeper().SetTokenPair(ctx, expPair)
				suite.network.App.GetErc20Keeper().SetDenomMap(ctx, expPair.Denom, id)
				suite.network.App.GetErc20Keeper().SetERC20Map(ctx, expPair.GetERC20Contract(), id)

				receiver = sdk.AccAddress(utiltx.GenerateAddress().Bytes())
			},
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset
			ctx = suite.network.GetContext()

			tc.malleate()

			pair, err := suite.network.App.GetErc20Keeper().MintingEnabled(ctx, sender, receiver, expPair.Erc20Address)
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(expPair, pair)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *PrecompileTestSuite) TestMintCoins() {
	var ctx sdk.Context
	sender := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	to := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	expPair := types.NewTokenPair(utiltx.GenerateAddress(), "coin", types.OWNER_MODULE)
	expPair.SetOwnerAddress(sender.String())
	amount := big.NewInt(1000000)
	id := expPair.GetID()

	testcases := []struct {
		name        string
		malleate    func()
		postCheck   func()
		expErr      bool
		errContains string
	}{
		{
			"fail - conversion is disabled globally",
			func() {
				params := types.DefaultParams()
				params.EnableErc20 = false
				suite.network.App.GetErc20Keeper().SetParams(ctx, params) //nolint:errcheck
			},
			func() {},
			true,
			"",
		},
		{
			"fail - token pair not found",
			func() {},
			func() {},
			true,
			"",
		},
		{
			"fail - conversion is disabled for the given pair",
			func() {
				expPair.Enabled = false
				suite.network.App.GetErc20Keeper().SetTokenPair(ctx, expPair)
				suite.network.App.GetErc20Keeper().SetDenomMap(ctx, expPair.Denom, id)
				suite.network.App.GetErc20Keeper().SetERC20Map(ctx, expPair.GetERC20Contract(), id)
			},
			func() {},
			true,
			"",
		},
		{
			"fail - token transfers are disabled",
			func() {
				expPair.Enabled = true
				suite.network.App.GetErc20Keeper().SetTokenPair(ctx, expPair)
				suite.network.App.GetErc20Keeper().SetDenomMap(ctx, expPair.Denom, id)
				suite.network.App.GetErc20Keeper().SetERC20Map(ctx, expPair.GetERC20Contract(), id)

				params := banktypes.DefaultParams()
				params.SendEnabled = []*banktypes.SendEnabled{
					{Denom: expPair.Denom, Enabled: false},
				}
				err := suite.network.App.GetBankKeeper().SetParams(ctx, params)
				suite.Require().NoError(err)
			},
			func() {},
			true,
			"",
		},
		{
			"fail - token not registered",
			func() {
				suite.network.App.GetErc20Keeper().SetDenomMap(ctx, expPair.Denom, id)
				suite.network.App.GetErc20Keeper().SetERC20Map(ctx, expPair.GetERC20Contract(), id)
			},
			func() {},
			true,
			"",
		},
		{
			"fail - receiver address is blocked (module account)",
			func() {
				suite.network.App.GetErc20Keeper().SetTokenPair(ctx, expPair)
				suite.network.App.GetErc20Keeper().SetDenomMap(ctx, expPair.Denom, id)
				suite.network.App.GetErc20Keeper().SetERC20Map(ctx, expPair.GetERC20Contract(), id)

				acc := suite.network.App.GetAccountKeeper().GetModuleAccount(ctx, types.ModuleName)
				to = acc.GetAddress()
			},
			func() {},
			true,
			"",
		},
		{
			"fail - pair is not native coin",
			func() {
				expPair.ContractOwner = types.OWNER_EXTERNAL
				suite.network.App.GetErc20Keeper().SetTokenPair(ctx, expPair)
				suite.network.App.GetErc20Keeper().SetDenomMap(ctx, expPair.Denom, id)
				suite.network.App.GetErc20Keeper().SetERC20Map(ctx, expPair.GetERC20Contract(), id)

				to = sdk.AccAddress(utiltx.GenerateAddress().Bytes())
			},
			func() {},
			true,
			types.ErrNonNativeCoinMintingDisabled.Error(),
		},
		{
			"fail - minter is not the owner",
			func() {
				expPair.ContractOwner = types.OWNER_MODULE
				expPair.SetOwnerAddress(sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String())
				suite.network.App.GetErc20Keeper().SetTokenPair(ctx, expPair)
				suite.network.App.GetErc20Keeper().SetDenomMap(ctx, expPair.Denom, id)
				suite.network.App.GetErc20Keeper().SetERC20Map(ctx, expPair.GetERC20Contract(), id)
			},
			func() {},
			true,
			types.ErrMinterIsNotOwner.Error(),
		},
		{
			"pass",
			func() {
				expPair.SetOwnerAddress(sender.String())
				suite.network.App.GetErc20Keeper().SetTokenPair(ctx, expPair)
				suite.network.App.GetErc20Keeper().SetDenomMap(ctx, expPair.Denom, id)
				suite.network.App.GetErc20Keeper().SetERC20Map(ctx, expPair.GetERC20Contract(), id)

				to = sdk.AccAddress(utiltx.GenerateAddress().Bytes())
			},
			func() {},
			false,
			"",
		},
	}

	for _, tc := range testcases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			ctx = suite.network.GetContext()

			tc.malleate()

			err := suite.network.App.GetErc20Keeper().MintCoins(ctx, sender, to, math.NewIntFromBigInt(amount), expPair.Erc20Address)
			if tc.expErr {
				suite.Require().Error(err, "expected transfer transaction to fail")
				suite.Require().Contains(err.Error(), tc.errContains, "expected transfer transaction to fail with specific error")
			} else {
				suite.Require().NoError(err, "expected transfer transaction succeeded")
				tc.postCheck()
			}
		})
	}
}
