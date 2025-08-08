package keeper_test

import (
	utiltx "github.com/cosmos/evm/testutil/tx"
	"github.com/cosmos/evm/x/erc20/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *KeeperTestSuite) TestTransferOwnership() {
	var ctx sdk.Context
	sender := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	newOwner := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	expPair := types.NewTokenPair(utiltx.GenerateAddress(), "coin", types.OWNER_MODULE)
	expPair.SetOwnerAddress(sender.String())
	id := expPair.GetID()

	testcases := []struct {
		name        string
		malleate    func()
		postCheck   func()
		expErr      bool
		errContains string
	}{
		{
			"fail - token pair not found",
			func() {
				params := types.DefaultParams()
				params.EnableErc20 = true
				suite.network.App.Erc20Keeper.SetParams(ctx, params) //nolint:errcheck
			},
			func() {},
			true,
			"",
		},
		{
			"fail - pair is not native coin",
			func() {
				expPair.ContractOwner = types.OWNER_EXTERNAL
				expPair.SetOwnerAddress(sender.String())
				suite.network.App.Erc20Keeper.SetTokenPair(ctx, expPair)
				suite.network.App.Erc20Keeper.SetDenomMap(ctx, expPair.Denom, id)
				suite.network.App.Erc20Keeper.SetERC20Map(ctx, expPair.GetERC20Contract(), id)
			},
			func() {},
			true,
			types.ErrNonNativeTransferOwnershipDisabled.Error(),
		},
		{
			"pass",
			func() {
				expPair.ContractOwner = types.OWNER_MODULE
				expPair.SetOwnerAddress(sender.String())
				suite.network.App.Erc20Keeper.SetTokenPair(ctx, expPair)
				suite.network.App.Erc20Keeper.SetDenomMap(ctx, expPair.Denom, id)
				suite.network.App.Erc20Keeper.SetERC20Map(ctx, expPair.GetERC20Contract(), id)
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

			err := suite.network.App.Erc20Keeper.TransferOwnership(ctx, sender, newOwner, expPair.Erc20Address)
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
