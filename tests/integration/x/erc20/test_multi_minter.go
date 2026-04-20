package erc20

import (
	"strings"

	utiltx "github.com/cosmos/evm/testutil/tx"
	"github.com/cosmos/evm/x/erc20/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *KeeperTestSuite) registerPair(ctx sdk.Context, pair types.TokenPair) {
	s.network.App.GetErc20Keeper().SetTokenPair(ctx, pair)
	s.network.App.GetErc20Keeper().SetDenomMap(ctx, pair.Denom, pair.GetID())
	s.network.App.GetErc20Keeper().SetERC20Map(ctx, pair.GetERC20Contract(), pair.GetID())
}

func (s *KeeperTestSuite) requireOwnerAddressesEvent(ctx sdk.Context, action, token, minter string, ownerAddresses []string) {
	events := ctx.EventManager().Events()
	ev := events[len(events)-1]
	attrs := make(map[string]string, len(ev.Attributes))
	for _, a := range ev.Attributes {
		attrs[a.Key] = a.Value
	}
	s.Require().Equal(action, attrs[sdk.AttributeKeyAction])
	s.Require().Equal(types.ModuleName, attrs[sdk.AttributeKeyModule])
	s.Require().Equal(token, attrs[types.AttributeKeyToken])
	s.Require().Equal(minter, attrs[types.AttributeKeyMinterAddress])
	s.Require().Equal(strings.Join(ownerAddresses, ","), attrs[types.AttributeKeyOwnerAddresses])
}

func (s *KeeperTestSuite) TestAddMinterAddress() {
	var (
		ctx   sdk.Context
		pair  types.TokenPair
		token string
	)
	minter := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	other := sdk.AccAddress(utiltx.GenerateAddress().Bytes())

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"token pair not found",
			func() {
				token = utiltx.GenerateAddress().String()
			},
			types.ErrTokenPairNotFound,
		},
		{
			"external token not supported",
			func() {
				pair = types.NewTokenPair(utiltx.GenerateAddress(), "coin", types.OWNER_EXTERNAL)
				s.registerPair(ctx, pair)
				token = pair.Erc20Address
			},
			types.ErrExternalTokenNotSupported,
		},
		{
			"minter already exists",
			func() {
				pair = types.NewTokenPair(utiltx.GenerateAddress(), "coin", types.OWNER_MODULE)
				pair.OwnerAddresses = []string{minter.String()}
				s.registerPair(ctx, pair)
				token = pair.Erc20Address
			},
			types.ErrMinterAlreadyExists,
		},
		{
			"add new minter to existing list",
			func() {
				pair = types.NewTokenPair(utiltx.GenerateAddress(), "coin", types.OWNER_MODULE)
				pair.OwnerAddresses = []string{other.String()}
				s.registerPair(ctx, pair)
				token = pair.Erc20Address
			},
			nil,
		},
		{
			"add new minter to empty list",
			func() {
				pair = types.NewTokenPair(utiltx.GenerateAddress(), "coin", types.OWNER_MODULE)
				s.registerPair(ctx, pair)
				token = pair.Erc20Address
			},
			nil,
		},
		{
			"add new minter - search by denom",
			func() {
				pair = types.NewTokenPair(utiltx.GenerateAddress(), "denomlookup", types.OWNER_MODULE)
				s.registerPair(ctx, pair)
				token = pair.Denom
			},
			nil,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			ctx = s.network.GetContext()
			pair = types.TokenPair{}
			token = ""

			tc.malleate()

			err := s.network.App.GetErc20Keeper().AddMinterAddress(ctx, minter, token)
			if tc.expErr != nil {
				s.Require().ErrorIs(err, tc.expErr)
				return
			}
			s.Require().NoError(err)

			stored, found := s.network.App.GetErc20Keeper().GetTokenPair(ctx, pair.GetID())
			s.Require().True(found)
			s.Require().Contains(stored.OwnerAddresses, minter.String())

			s.requireOwnerAddressesEvent(ctx, types.TypeMsgAddMinter, token, minter.String(), stored.OwnerAddresses)
		})
	}
}

func (s *KeeperTestSuite) TestRemoveMinterAddress() {
	var (
		ctx   sdk.Context
		pair  types.TokenPair
		token string
	)
	minter := sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	other := sdk.AccAddress(utiltx.GenerateAddress().Bytes())

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"token pair not found",
			func() {
				token = utiltx.GenerateAddress().String()
			},
			types.ErrTokenPairNotFound,
		},
		{
			"minter not found",
			func() {
				pair = types.NewTokenPair(utiltx.GenerateAddress(), "coin", types.OWNER_MODULE)
				pair.OwnerAddresses = []string{other.String()}
				s.registerPair(ctx, pair)
				token = pair.Erc20Address
			},
			types.ErrMinterNotFound,
		},
		{
			"cannot remove last minter",
			func() {
				pair = types.NewTokenPair(utiltx.GenerateAddress(), "coin", types.OWNER_MODULE)
				pair.OwnerAddresses = []string{minter.String()}
				s.registerPair(ctx, pair)
				token = pair.Erc20Address
			},
			types.ErrCannotRemoveLastMinter,
		},
		{
			"remove minter from list",
			func() {
				pair = types.NewTokenPair(utiltx.GenerateAddress(), "coin", types.OWNER_MODULE)
				pair.OwnerAddresses = []string{minter.String(), other.String()}
				s.registerPair(ctx, pair)
				token = pair.Erc20Address
			},
			nil,
		},
		{
			"remove minter - search by denom",
			func() {
				pair = types.NewTokenPair(utiltx.GenerateAddress(), "denomlookup", types.OWNER_MODULE)
				pair.OwnerAddresses = []string{minter.String(), other.String()}
				s.registerPair(ctx, pair)
				token = pair.Denom
			},
			nil,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			ctx = s.network.GetContext()
			pair = types.TokenPair{}
			token = ""

			tc.malleate()

			err := s.network.App.GetErc20Keeper().RemoveMinterAddress(ctx, minter, token)
			if tc.expErr != nil {
				s.Require().ErrorIs(err, tc.expErr)
				return
			}
			s.Require().NoError(err)

			stored, found := s.network.App.GetErc20Keeper().GetTokenPair(ctx, pair.GetID())
			s.Require().True(found)
			s.Require().NotContains(stored.OwnerAddresses, minter.String())

			s.requireOwnerAddressesEvent(ctx, types.TypeMsgRemoveMinter, token, minter.String(), stored.OwnerAddresses)
		})
	}
}

func (s *KeeperTestSuite) TestGetOwnerAddresses() {
	s.SetupTest()
	ctx := s.network.GetContext()

	s.Run("unknown contract returns empty", func() {
		addrs := s.network.App.GetErc20Keeper().GetOwnerAddresses(ctx, utiltx.GenerateAddress().String())
		s.Require().Empty(addrs)
	})

	s.Run("external contract returns empty", func() {
		pair := types.NewTokenPair(utiltx.GenerateAddress(), "external", types.OWNER_EXTERNAL)
		s.registerPair(ctx, pair)

		addrs := s.network.App.GetErc20Keeper().GetOwnerAddresses(ctx, pair.Erc20Address)
		s.Require().Empty(addrs)
	})

	s.Run("known contract returns OwnerAddresses", func() {
		owner := sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String()
		pair := types.NewTokenPair(utiltx.GenerateAddress(), "coin", types.OWNER_MODULE)
		pair.OwnerAddresses = []string{owner}
		s.registerPair(ctx, pair)

		addrs := s.network.App.GetErc20Keeper().GetOwnerAddresses(ctx, pair.Erc20Address)
		s.Require().Equal([]string{owner}, addrs)
	})
}

func (s *KeeperTestSuite) TestMigrateOwnerAddresses() {
	s.SetupTest()
	ctx := s.network.GetContext()
	k := s.network.App.GetErc20Keeper()

	legacyOwner := sdk.AccAddress(utiltx.GenerateAddress().Bytes()).String()

	migrate := types.NewTokenPair(utiltx.GenerateAddress(), "migrate", types.OWNER_MODULE)
	migrate.OwnerAddress = legacyOwner
	k.SetTokenPair(ctx, migrate)

	ibcLegacy := types.NewTokenPair(utiltx.GenerateAddress(), "empty", types.OWNER_MODULE)
	k.SetTokenPair(ctx, ibcLegacy)

	externalLegacy := types.NewTokenPair(utiltx.GenerateAddress(), "empty", types.OWNER_EXTERNAL)
	k.SetTokenPair(ctx, externalLegacy)

	// Impossible state by validation rules, asserts the migration guard checks
	// ContractOwner == OWNER_MODULE, not just OwnerAddress != "".
	externalWithLegacy := types.NewTokenPair(utiltx.GenerateAddress(), "external-legacy", types.OWNER_EXTERNAL)
	externalWithLegacy.OwnerAddress = legacyOwner
	k.SetTokenPair(ctx, externalWithLegacy)

	k.MigrateOwnerAddresses(ctx)

	got, found := k.GetTokenPair(ctx, migrate.GetID())
	s.Require().True(found)
	s.Require().Equal([]string{legacyOwner}, got.OwnerAddresses)
	s.Require().Equal("", got.OwnerAddress)

	got, found = k.GetTokenPair(ctx, ibcLegacy.GetID())
	s.Require().True(found)
	s.Require().Empty(got.OwnerAddresses)
	s.Require().Empty(got.OwnerAddress)

	got, found = k.GetTokenPair(ctx, externalLegacy.GetID())
	s.Require().True(found)
	s.Require().Empty(got.OwnerAddresses)
	s.Require().Empty(got.OwnerAddress)

	got, found = k.GetTokenPair(ctx, externalWithLegacy.GetID())
	s.Require().True(found)
	s.Require().Empty(got.OwnerAddresses)
	s.Require().Equal(legacyOwner, got.OwnerAddress)
}
