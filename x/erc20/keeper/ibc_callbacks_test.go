package keeper_test

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/cosmos/evm/contracts"
	"github.com/cosmos/evm/crypto/ethsecp256k1"
	"github.com/cosmos/evm/testutil"
	"github.com/cosmos/evm/utils"
	"github.com/cosmos/evm/x/erc20/keeper"
	"github.com/cosmos/evm/x/erc20/types"
	"github.com/cosmos/evm/x/vm/statedb"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibcgotesting "github.com/cosmos/ibc-go/v10/testing"
	ibcmock "github.com/cosmos/ibc-go/v10/testing/mock"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

var erc20Denom = "erc20:0xdac17f958d2ee523a2206206994597c13d831ec7"

func (suite *KeeperTestSuite) TestOnRecvPacketRegistered() {
	var ctx sdk.Context
	// secp256k1 account
	secpPk := secp256k1.GenPrivKey()
	secpAddr := sdk.AccAddress(secpPk.PubKey().Address())
	secpAddrCosmos := sdk.MustBech32ifyAddressBytes(sdk.Bech32MainPrefix, secpAddr)

	// ethsecp256k1 account
	ethPk, err := ethsecp256k1.GenerateKey()
	suite.Require().Nil(err)
	ethsecpAddr := sdk.AccAddress(ethPk.PubKey().Address())
	ethsecpAddrEvmos := sdk.AccAddress(ethPk.PubKey().Address()).String()
	ethsecpAddrCosmos := sdk.MustBech32ifyAddressBytes(sdk.Bech32MainPrefix, ethsecpAddr)

	// Setup Cosmos <=> Cosmos EVM IBC relayer
	sourceChannel := "channel-292"
	cosmosEVMChannel := "channel-3"
	hop := transfertypes.NewHop(transfertypes.PortID, cosmosEVMChannel)

	timeoutHeight := clienttypes.NewHeight(0, 100)
	disabledTimeoutTimestamp := uint64(0)
	mockPacket := channeltypes.NewPacket(ibcgotesting.MockPacketData, 1, transfertypes.PortID, "channel-0", transfertypes.PortID, "channel-0", timeoutHeight, disabledTimeoutTimestamp)
	packet := mockPacket
	expAck := ibcmock.MockAcknowledgement

	baseDenom, err := sdk.GetBaseDenom()
	suite.Require().NoError(err, "failed to get base denom")
	registeredDenom := cosmosTokenBase
	coins := sdk.NewCoins(
		sdk.NewCoin(baseDenom, math.NewInt(1000)),
		sdk.NewCoin(registeredDenom, math.NewInt(1000)), // some ERC20 token
		sdk.NewCoin(ibcBase, math.NewInt(1000)),         // some IBC coin with a registered token pair
	)

	testCases := []struct {
		name             string
		malleate         func()
		ackSuccess       bool
		receiver         sdk.AccAddress
		expErc20s        *big.Int
		expCoins         sdk.Coins
		checkBalances    bool
		disableERC20     bool
		disableTokenPair bool
	}{
		{
			name: "error - non ics-20 packet",
			malleate: func() {
				packet = mockPacket
			},
			receiver:      secpAddr,
			ackSuccess:    false,
			checkBalances: false,
			expErc20s:     big.NewInt(0),
			expCoins:      coins,
		},
		{
			name: "no-op - erc20 module param disabled",
			malleate: func() {
				transfer := transfertypes.NewFungibleTokenPacketData(registeredDenom, "100", ethsecpAddrEvmos, ethsecpAddrCosmos, "")
				bz := transfertypes.ModuleCdc.MustMarshalJSON(&transfer)
				packet = channeltypes.NewPacket(bz, 1, transfertypes.PortID, sourceChannel, transfertypes.PortID, cosmosEVMChannel, timeoutHeight, 0)
			},
			receiver:      secpAddr,
			disableERC20:  true,
			ackSuccess:    true,
			checkBalances: false,
			expErc20s:     big.NewInt(0),
			expCoins:      coins,
		},
		{
			name: "error - invalid sender (no '1')",
			malleate: func() {
				transfer := transfertypes.NewFungibleTokenPacketData(registeredDenom, "100", "evmos", ethsecpAddrCosmos, "")
				bz := transfertypes.ModuleCdc.MustMarshalJSON(&transfer)
				packet = channeltypes.NewPacket(bz, 100, transfertypes.PortID, sourceChannel, transfertypes.PortID, cosmosEVMChannel, timeoutHeight, 0)
			},
			receiver:      secpAddr,
			ackSuccess:    false,
			checkBalances: false,
			expErc20s:     big.NewInt(0),
			expCoins:      coins,
		},
		{
			name: "error - invalid sender (bad address)",
			malleate: func() {
				transfer := transfertypes.NewFungibleTokenPacketData(registeredDenom, "100", "badba1sv9m0g7ycejwr3s369km58h5qe7xj77hvcxrms", ethsecpAddrCosmos, "")
				bz := transfertypes.ModuleCdc.MustMarshalJSON(&transfer)
				packet = channeltypes.NewPacket(bz, 100, transfertypes.PortID, sourceChannel, transfertypes.PortID, cosmosEVMChannel, timeoutHeight, 0)
			},
			receiver:      secpAddr,
			ackSuccess:    false,
			checkBalances: false,
			expErc20s:     big.NewInt(0),
			expCoins:      coins,
		},
		{
			name: "error - invalid recipient (bad address)",
			malleate: func() {
				transfer := transfertypes.NewFungibleTokenPacketData(registeredDenom, "100", ethsecpAddrEvmos, "badbadhf0468jjpe6m6vx38s97z2qqe8ldu0njdyf625", "")
				bz := transfertypes.ModuleCdc.MustMarshalJSON(&transfer)
				packet = channeltypes.NewPacket(bz, 100, transfertypes.PortID, sourceChannel, transfertypes.PortID, cosmosEVMChannel, timeoutHeight, 0)
			},
			receiver:      secpAddr,
			ackSuccess:    false,
			checkBalances: false,
			expErc20s:     big.NewInt(0),
			expCoins:      coins,
		},
		{
			name: "no-op - receiver is module account",
			malleate: func() {
				secpAddr = suite.network.App.AccountKeeper.GetModuleAccount(ctx, "erc20").GetAddress()
				transfer := transfertypes.NewFungibleTokenPacketData(registeredDenom, "100", secpAddrCosmos, secpAddr.String(), "")
				bz := transfertypes.ModuleCdc.MustMarshalJSON(&transfer)
				packet = channeltypes.NewPacket(bz, 100, transfertypes.PortID, sourceChannel, transfertypes.PortID, cosmosEVMChannel, timeoutHeight, 0)
			},
			ackSuccess:    true,
			receiver:      secpAddr,
			expErc20s:     big.NewInt(0),
			expCoins:      coins,
			checkBalances: true,
		},
		{
			name: "no-op - base denomination",
			malleate: func() {
				// base denom should be prefixed
				hop := transfertypes.NewHop(transfertypes.PortID, sourceChannel)
				bondDenom, err := suite.network.App.StakingKeeper.BondDenom(ctx)
				suite.Require().NoError(err)
				prefixedDenom := transfertypes.NewDenom(bondDenom, hop).Path()
				transfer := transfertypes.NewFungibleTokenPacketData(prefixedDenom, "100", secpAddrCosmos, ethsecpAddrEvmos, "")
				bz := transfertypes.ModuleCdc.MustMarshalJSON(&transfer)
				packet = channeltypes.NewPacket(bz, 1, transfertypes.PortID, sourceChannel, transfertypes.PortID, cosmosEVMChannel, timeoutHeight, 0)
			},
			ackSuccess:    true,
			receiver:      ethsecpAddr,
			expErc20s:     big.NewInt(0),
			expCoins:      coins,
			checkBalances: true,
		},
		{
			name: "no-op - pair is not registered",
			malleate: func() {
				transfer := transfertypes.NewFungibleTokenPacketData(erc20Denom, "100", secpAddrCosmos, ethsecpAddrEvmos, "")
				bz := transfertypes.ModuleCdc.MustMarshalJSON(&transfer)
				packet = channeltypes.NewPacket(bz, 1, transfertypes.PortID, sourceChannel, transfertypes.PortID, cosmosEVMChannel, timeoutHeight, 0)
			},
			ackSuccess:    true,
			receiver:      ethsecpAddr,
			expErc20s:     big.NewInt(0),
			expCoins:      coins,
			checkBalances: true,
		},
		{
			name: "error - pair is not registered but erc20 registered",
			malleate: func() {
				transfer := transfertypes.NewFungibleTokenPacketData(erc20Denom, "100", secpAddrCosmos, ethsecpAddrEvmos, "")
				bz := transfertypes.ModuleCdc.MustMarshalJSON(&transfer)
				packet = channeltypes.NewPacket(bz, 1, transfertypes.PortID, sourceChannel, transfertypes.PortID, cosmosEVMChannel, timeoutHeight, 0)
				collidedAddr, err := utils.GetIBCDenomAddress(transfertypes.NewDenom(erc20Denom, hop).IBCDenom())
				suite.Require().NoError(err)
				suite.network.App.Erc20Keeper.SetERC20Map(ctx, collidedAddr, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
				suite.Require().True(suite.network.App.Erc20Keeper.IsERC20Registered(ctx, collidedAddr))
			},
			ackSuccess:    false,
			receiver:      secpAddr,
			expErc20s:     big.NewInt(0),
			expCoins:      coins,
			checkBalances: false,
		},
		{
			name: "error - pair is not registered but denom registered",
			malleate: func() {
				transfer := transfertypes.NewFungibleTokenPacketData(erc20Denom, "100", secpAddrCosmos, ethsecpAddrEvmos, "")
				bz := transfertypes.ModuleCdc.MustMarshalJSON(&transfer)
				packet = channeltypes.NewPacket(bz, 1, transfertypes.PortID, sourceChannel, transfertypes.PortID, cosmosEVMChannel, timeoutHeight, 0)
				collidedDenom := transfertypes.NewDenom(erc20Denom, hop).IBCDenom()
				suite.network.App.Erc20Keeper.SetDenomMap(ctx, collidedDenom, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
				suite.Require().True(suite.network.App.Erc20Keeper.IsDenomRegistered(ctx, collidedDenom))
			},
			ackSuccess:    false,
			receiver:      secpAddr,
			expErc20s:     big.NewInt(0),
			expCoins:      coins,
			checkBalances: false,
		},
		{
			name: "error - pair is not registered but address has code",
			malleate: func() {
				transfer := transfertypes.NewFungibleTokenPacketData(erc20Denom, "100", secpAddrCosmos, ethsecpAddrEvmos, "")
				bz := transfertypes.ModuleCdc.MustMarshalJSON(&transfer)
				packet = channeltypes.NewPacket(bz, 1, transfertypes.PortID, sourceChannel, transfertypes.PortID, cosmosEVMChannel, timeoutHeight, 0)
				collidedAddr, err := utils.GetIBCDenomAddress(transfertypes.NewDenom(erc20Denom, hop).IBCDenom())
				suite.Require().NoError(err)
				suite.Require().False(suite.network.App.Erc20Keeper.IsERC20Registered(ctx, collidedAddr))
				err = suite.network.App.EVMKeeper.SetAccount(ctx, collidedAddr, statedb.Account{
					Nonce:    0,
					Balance:  nil,
					CodeHash: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
				})
				suite.Require().NoError(err)
				suite.Require().True(suite.network.App.EVMKeeper.IsContract(ctx, collidedAddr))
			},
			ackSuccess:    false,
			receiver:      secpAddr,
			expErc20s:     big.NewInt(0),
			expCoins:      coins,
			checkBalances: false,
		},
		{
			name: "no-op - pair disabled",
			malleate: func() {
				pk1 := secp256k1.GenPrivKey()
				hop := transfertypes.NewHop(transfertypes.PortID, sourceChannel)
				prefixedDenom := transfertypes.NewDenom(registeredDenom, hop).Path()
				otherSecpAddrEvmos := sdk.AccAddress(pk1.PubKey().Address()).String()
				transfer := transfertypes.NewFungibleTokenPacketData(prefixedDenom, "500", otherSecpAddrEvmos, ethsecpAddrEvmos, "")
				bz := transfertypes.ModuleCdc.MustMarshalJSON(&transfer)
				packet = channeltypes.NewPacket(bz, 1, transfertypes.PortID, sourceChannel, transfertypes.PortID, cosmosEVMChannel, timeoutHeight, 0)
			},
			ackSuccess: true,
			receiver:   ethsecpAddr,
			expErc20s:  big.NewInt(0),
			expCoins: sdk.NewCoins(
				sdk.NewCoin(baseDenom, math.NewInt(1000)),
				sdk.NewCoin(registeredDenom, math.NewInt(0)),
				sdk.NewCoin(ibcBase, math.NewInt(1000)),
			),
			checkBalances:    false,
			disableTokenPair: true,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.mintFeeCollector = true
			suite.SetupTest() // reset
			ctx = suite.network.GetContext()

			// Register Token Pair for testing
			contractAddr, err := suite.setupRegisterERC20Pair(contractMinterBurner)
			suite.Require().NoError(err, "failed to register pair")
			// get updated context after registering ERC20 pair
			ctx = suite.network.GetContext()

			// Set Denom
			denom := transfertypes.NewDenom(registeredDenom, hop)
			suite.network.App.TransferKeeper.SetDenom(ctx, denom)

			// Set Cosmos Channel
			channel := channeltypes.Channel{
				State:          channeltypes.INIT,
				Ordering:       channeltypes.UNORDERED,
				Counterparty:   channeltypes.NewCounterparty(transfertypes.PortID, sourceChannel),
				ConnectionHops: []string{sourceChannel},
			}
			suite.network.App.IBCKeeper.ChannelKeeper.SetChannel(ctx, transfertypes.PortID, cosmosEVMChannel, channel)

			// Set Next Sequence Send
			suite.network.App.IBCKeeper.ChannelKeeper.SetNextSequenceSend(ctx, transfertypes.PortID, cosmosEVMChannel, 1)

			suite.network.App.Erc20Keeper = keeper.NewKeeper(
				suite.network.App.GetKey(types.StoreKey),
				suite.network.App.AppCodec(),
				authtypes.NewModuleAddress(govtypes.ModuleName),
				suite.network.App.AccountKeeper,
				suite.network.App.BankKeeper,
				suite.network.App.EVMKeeper,
				suite.network.App.StakingKeeper,
				&suite.network.App.TransferKeeper,
			)

			// Fund receiver account with ATOM, ERC20 coins and IBC vouchers
			// We do this since we are interested in the conversion portion w/ OnRecvPacket
			err = testutil.FundAccount(ctx, suite.network.App.BankKeeper, tc.receiver, coins)
			suite.Require().NoError(err)

			id := suite.network.App.Erc20Keeper.GetTokenPairID(ctx, contractAddr.String())
			pair, _ := suite.network.App.Erc20Keeper.GetTokenPair(ctx, id)
			suite.Require().NotNil(pair)

			if tc.disableERC20 {
				params := suite.network.App.Erc20Keeper.GetParams(ctx)
				params.EnableErc20 = false
				suite.network.App.Erc20Keeper.SetParams(ctx, params) //nolint:errcheck
			}

			if tc.disableTokenPair {
				_, err := suite.network.App.Erc20Keeper.ToggleConversion(ctx, &types.MsgToggleConversion{
					Authority: authtypes.NewModuleAddress("gov").String(),
					Token:     pair.Denom,
				})
				suite.Require().NoError(err)
			}

			tc.malleate()

			// Perform IBC callback
			ack := suite.network.App.Erc20Keeper.OnRecvPacket(ctx, packet, expAck)

			// Check acknowledgement
			if tc.ackSuccess {
				suite.Require().True(ack.Success(), string(ack.Acknowledgement()))
				suite.Require().Equal(expAck, ack)
			} else {
				suite.Require().False(ack.Success(), string(ack.Acknowledgement()))
			}

			if tc.checkBalances {
				// Check ERC20 balances
				balanceTokenAfter := suite.network.App.Erc20Keeper.BalanceOf(ctx, contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(tc.receiver.Bytes()))
				suite.Require().Equal(tc.expErc20s.Int64(), balanceTokenAfter.Int64())
				// Check Cosmos Coin Balances
				balances := suite.network.App.BankKeeper.GetAllBalances(ctx, tc.receiver)
				suite.Require().Equal(tc.expCoins, balances)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestConvertCoinToERC20FromPacket() {
	var ctx sdk.Context
	senderAddr := "cosmos1x2w87cvt5mqjncav4lxy8yfreynn273x34qlwy"

	baseDenom, err := sdk.GetBaseDenom()
	suite.Require().NoError(err)

	testCases := []struct {
		name     string
		malleate func() transfertypes.FungibleTokenPacketData
		transfer transfertypes.FungibleTokenPacketData
		expPass  bool
	}{
		{
			name: "error - invalid sender",
			malleate: func() transfertypes.FungibleTokenPacketData {
				return transfertypes.NewFungibleTokenPacketData(baseDenom, "10", "", "", "")
			},
			expPass: false,
		},
		{
			name: "pass - is base denom",
			malleate: func() transfertypes.FungibleTokenPacketData {
				return transfertypes.NewFungibleTokenPacketData(baseDenom, "10", senderAddr, "", "")
			},
			expPass: true,
		},
		{
			name: "pass - erc20 is disabled",
			malleate: func() transfertypes.FungibleTokenPacketData {
				// Register Token Pair for testing
				contractAddr, err := suite.setupRegisterERC20Pair(contractMinterBurner)
				suite.Require().NoError(err, "failed to register pair")
				ctx = suite.network.GetContext()
				id := suite.network.App.Erc20Keeper.GetTokenPairID(ctx, contractAddr.String())
				pair, _ := suite.network.App.Erc20Keeper.GetTokenPair(ctx, id)
				suite.Require().NotNil(pair)

				params := suite.network.App.Erc20Keeper.GetParams(ctx)
				params.EnableErc20 = false
				_ = suite.network.App.Erc20Keeper.SetParams(ctx, params)
				return transfertypes.NewFungibleTokenPacketData(pair.Denom, "10", senderAddr, "", "")
			},
			expPass: true,
		},
		{
			name: "pass - denom is not registered",
			malleate: func() transfertypes.FungibleTokenPacketData {
				return transfertypes.NewFungibleTokenPacketData(metadataIbc.Base, "10", senderAddr, "", "")
			},
			expPass: true,
		},
		{
			name: "pass - erc20 is disabled",
			malleate: func() transfertypes.FungibleTokenPacketData {
				// Register Token Pair for testing
				contractAddr, err := suite.setupRegisterERC20Pair(contractMinterBurner)
				suite.Require().NoError(err, "failed to register pair")
				ctx = suite.network.GetContext()
				id := suite.network.App.Erc20Keeper.GetTokenPairID(ctx, contractAddr.String())
				pair, _ := suite.network.App.Erc20Keeper.GetTokenPair(ctx, id)
				suite.Require().NotNil(pair)

				err = testutil.FundAccount(
					ctx,
					suite.network.App.BankKeeper,
					sdk.MustAccAddressFromBech32(senderAddr),
					sdk.NewCoins(
						sdk.NewCoin(pair.Denom, math.NewInt(100)),
					),
				)
				suite.Require().NoError(err)

				_, err = suite.network.App.EVMKeeper.CallEVM(ctx, contracts.ERC20MinterBurnerDecimalsContract.ABI, suite.keyring.GetAddr(0), contractAddr, true, "mint", types.ModuleAddress, big.NewInt(10))
				suite.Require().NoError(err)

				return transfertypes.NewFungibleTokenPacketData(pair.Denom, "10", senderAddr, "", "")
			},
			expPass: true,
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.mintFeeCollector = true
			defer func() { suite.mintFeeCollector = false }()

			suite.SetupTest() // reset
			ctx = suite.network.GetContext()

			transfer := tc.malleate()

			err := suite.network.App.Erc20Keeper.ConvertCoinToERC20FromPacket(ctx, transfer)
			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestOnAcknowledgementPacket() {
	var (
		ctx  sdk.Context
		data transfertypes.FungibleTokenPacketData
		ack  channeltypes.Acknowledgement
		pair types.TokenPair
	)

	// secp256k1 account
	senderPk := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(senderPk.PubKey().Address())

	receiverPk := secp256k1.GenPrivKey()
	receiver := sdk.AccAddress(receiverPk.PubKey().Address())
	testCases := []struct {
		name           string
		malleate       func()
		expERC20       *big.Int
		expPass        bool
		expErrorEvents func()
	}{
		{
			name: "no-op - ack error sender is module account",
			malleate: func() {
				// Register Token Pair for testing
				contractAddr, err := suite.setupRegisterERC20Pair(contractMinterBurner)
				suite.Require().NoError(err, "failed to register pair")
				ctx = suite.network.GetContext()
				id := suite.network.App.Erc20Keeper.GetTokenPairID(ctx, contractAddr.String())
				pair, _ = suite.network.App.Erc20Keeper.GetTokenPair(ctx, id)
				suite.Require().NotNil(pair)

				// for testing purposes we can only fund is not allowed to receive funds
				moduleAcc := suite.network.App.AccountKeeper.GetModuleAccount(ctx, "erc20")
				sender = moduleAcc.GetAddress()
				err = testutil.FundModuleAccount(
					ctx,
					suite.network.App.BankKeeper,
					moduleAcc.GetName(),
					sdk.NewCoins(
						sdk.NewCoin(pair.Denom, math.NewInt(100)),
					),
				)
				suite.Require().NoError(err)

				ack = channeltypes.NewErrorAcknowledgement(errors.New(""))
				data = transfertypes.NewFungibleTokenPacketData(pair.Denom, "100", sender.String(), receiver.String(), "")
			},
			expPass:        true,
			expERC20:       big.NewInt(0),
			expErrorEvents: func() {},
		},
		{
			name: "no-op - positive ack",
			malleate: func() {
				// Register Token Pair for testing
				contractAddr, err := suite.setupRegisterERC20Pair(contractMinterBurner)
				suite.Require().NoError(err, "failed to register pair")
				ctx = suite.network.GetContext()
				id := suite.network.App.Erc20Keeper.GetTokenPairID(ctx, contractAddr.String())
				pair, _ = suite.network.App.Erc20Keeper.GetTokenPair(ctx, id)
				suite.Require().NotNil(pair)

				sender = sdk.AccAddress(senderPk.PubKey().Address())

				// Fund receiver account with ATOM, ERC20 coins and IBC vouchers
				// We do this since we are interested in the conversion portion w/ OnRecvPacket
				err = testutil.FundAccount(
					ctx,
					suite.network.App.BankKeeper,
					sender,
					sdk.NewCoins(
						sdk.NewCoin(pair.Denom, math.NewInt(100)),
					),
				)
				suite.Require().NoError(err)

				ack = channeltypes.NewResultAcknowledgement([]byte{1})
			},
			expERC20:       big.NewInt(0),
			expPass:        true,
			expErrorEvents: func() {},
		},
		{
			name: "convert - error ack",
			malleate: func() {
				// Register Token Pair for testing
				contractAddr, err := suite.setupRegisterERC20Pair(contractMinterBurner)
				suite.Require().NoError(err, "failed to register pair")
				ctx = suite.network.GetContext()
				id := suite.network.App.Erc20Keeper.GetTokenPairID(ctx, contractAddr.String())
				pair, _ = suite.network.App.Erc20Keeper.GetTokenPair(ctx, id)
				suite.Require().NotNil(pair)

				sender = sdk.AccAddress(senderPk.PubKey().Address())

				// Fund receiver account with ATOM, ERC20 coins and IBC vouchers
				// We do this since we are interested in the conversion portion w/ OnRecvPacket
				err = testutil.FundAccount(
					ctx,
					suite.network.App.BankKeeper,
					sender,
					sdk.NewCoins(
						sdk.NewCoin(pair.Denom, math.NewInt(100)),
					),
				)
				suite.Require().NoError(err)

				_, err = suite.network.App.EVMKeeper.CallEVM(ctx, contracts.ERC20MinterBurnerDecimalsContract.ABI, suite.keyring.GetAddr(0), contractAddr, true, "mint", types.ModuleAddress, big.NewInt(100))
				suite.Require().NoError(err)

				ack = channeltypes.NewErrorAcknowledgement(errors.New("error"))
				data = transfertypes.NewFungibleTokenPacketData(pair.Denom, "100", sender.String(), receiver.String(), "")
			},
			expERC20:       big.NewInt(100),
			expPass:        true,
			expErrorEvents: func() {},
		},
		{
			name: "err - self-destructed contract",
			malleate: func() {
				// Register Token Pair for testing
				contractAddr, err := suite.setupRegisterERC20Pair(contractMinterBurner)
				suite.Require().NoError(err, "failed to register pair")
				ctx = suite.network.GetContext()
				id := suite.network.App.Erc20Keeper.GetTokenPairID(ctx, contractAddr.String())
				pair, _ = suite.network.App.Erc20Keeper.GetTokenPair(ctx, id)
				suite.Require().NotNil(pair)

				// self destruct the token
				err = suite.network.App.EVMKeeper.DeleteAccount(suite.network.GetContext(), contractAddr)
				suite.Require().NoError(err)

				sender = sdk.AccAddress(senderPk.PubKey().Address())

				// Fund receiver account with ATOM, ERC20 coins and IBC vouchers
				// We do this since we are interested in the conversion portion w/ OnRecvPacket
				err = testutil.FundAccount(
					ctx,
					suite.network.App.BankKeeper,
					sender,
					sdk.NewCoins(
						sdk.NewCoin(pair.Denom, math.NewInt(100)),
					),
				)
				suite.Require().NoError(err)

				ack = channeltypes.NewErrorAcknowledgement(errors.New("error"))
				data = transfertypes.NewFungibleTokenPacketData(pair.Denom, "100", sender.String(), receiver.String(), "")
			},
			expERC20: big.NewInt(0),
			expPass:  false,
			expErrorEvents: func() {
				event := ctx.EventManager().Events()[len(ctx.EventManager().Events())-1]
				suite.Require().Equal(event.Type, types.EventTypeFailedConvertERC20)
			},
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest() // reset
			ctx = suite.network.GetContext()

			tc.malleate()

			err := suite.network.App.Erc20Keeper.OnAcknowledgementPacket(
				ctx, channeltypes.Packet{}, data, ack,
			)

			if tc.expPass {
				suite.Require().NoError(err)
				// check balance is the same as expected
				balance := suite.network.App.Erc20Keeper.BalanceOf(
					ctx, contracts.ERC20MinterBurnerDecimalsContract.ABI,
					pair.GetERC20Contract(),
					common.BytesToAddress(sender.Bytes()),
				)
				suite.Require().Equal(tc.expERC20.Int64(), balance.Int64())
			} else {
				tc.expErrorEvents()
			}
		})
	}
}

func (suite *KeeperTestSuite) TestOnTimeoutPacket() {
	var (
		ctx  sdk.Context
		data transfertypes.FungibleTokenPacketData
		pair types.TokenPair
	)
	senderPk := secp256k1.GenPrivKey()
	sender := sdk.AccAddress(senderPk.PubKey().Address())
	receiverPk := secp256k1.GenPrivKey()
	receiver := sdk.AccAddress(receiverPk.PubKey().Address())

	testCases := []struct {
		name           string
		malleate       func()
		expERC20       *big.Int
		expPass        bool
		expErrorEvents func()
	}{
		{
			name: "convert - pass timeout",
			malleate: func() {
				// Register Token Pair for testing
				contractAddr, err := suite.setupRegisterERC20Pair(contractMinterBurner)
				suite.Require().NoError(err, "failed to register pair")
				ctx = suite.network.GetContext()
				id := suite.network.App.Erc20Keeper.GetTokenPairID(ctx, contractAddr.String())
				pair, _ = suite.network.App.Erc20Keeper.GetTokenPair(ctx, id)
				suite.Require().NotNil(pair)

				_, err = suite.network.App.EVMKeeper.CallEVM(ctx, contracts.ERC20MinterBurnerDecimalsContract.ABI, suite.keyring.GetAddr(0), contractAddr, true, "mint", types.ModuleAddress, big.NewInt(100))
				suite.Require().NoError(err)

				// Fund module account with ATOM, ERC20 coins and IBC vouchers
				// We do this since we are interested in the conversion portion w/ OnRecvPacket
				err = testutil.FundAccount(
					ctx,
					suite.network.App.BankKeeper,
					sender,
					sdk.NewCoins(
						sdk.NewCoin(pair.Denom, math.NewInt(100)),
					),
				)
				suite.Require().NoError(err)

				data = transfertypes.NewFungibleTokenPacketData(pair.Denom, "10", sender.String(), receiver.String(), "")
			},
			expERC20:       big.NewInt(10),
			expPass:        true,
			expErrorEvents: func() {},
		},
		{
			name: "no-op - sender is module account",
			malleate: func() {
				// Register Token Pair for testing
				contractAddr, err := suite.setupRegisterERC20Pair(contractMinterBurner)
				suite.Require().NoError(err, "failed to register pair")
				ctx = suite.network.GetContext()
				id := suite.network.App.Erc20Keeper.GetTokenPairID(ctx, contractAddr.String())
				pair, _ = suite.network.App.Erc20Keeper.GetTokenPair(ctx, id)
				suite.Require().NotNil(pair)

				// any module account can be passed here
				moduleAcc := suite.network.App.AccountKeeper.GetModuleAccount(ctx, evmtypes.ModuleName)

				data = transfertypes.NewFungibleTokenPacketData(pair.Denom, "10", moduleAcc.GetAddress().String(), "", "")
			},
			expERC20:       big.NewInt(0),
			expPass:        true,
			expErrorEvents: func() {},
		},
		{
			name: "err - self-destructed contract",
			malleate: func() {
				// Register Token Pair for testing
				contractAddr, err := suite.setupRegisterERC20Pair(contractMinterBurner)
				suite.Require().NoError(err, "failed to register pair")
				ctx = suite.network.GetContext()
				id := suite.network.App.Erc20Keeper.GetTokenPairID(ctx, contractAddr.String())
				pair, _ = suite.network.App.Erc20Keeper.GetTokenPair(ctx, id)
				suite.Require().NotNil(pair)
				// self destruct the token
				err = suite.network.App.EVMKeeper.DeleteAccount(suite.network.GetContext(), contractAddr)
				suite.Require().NoError(err)

				// Fund receiver account with ATOM, ERC20 coins and IBC vouchers
				// We do this since we are interested in the conversion portion w/ OnRecvPacket
				err = testutil.FundAccount(
					ctx,
					suite.network.App.BankKeeper,
					sender,
					sdk.NewCoins(
						sdk.NewCoin(pair.Denom, math.NewInt(100)),
					),
				)
				suite.Require().NoError(err)

				data = transfertypes.NewFungibleTokenPacketData(pair.Denom, "100", sender.String(), receiver.String(), "")
			},
			expERC20: big.NewInt(0),
			expPass:  false,
			expErrorEvents: func() {
				event := ctx.EventManager().Events()[len(ctx.EventManager().Events())-1]
				suite.Require().Equal(event.Type, types.EventTypeFailedConvertERC20)
			},
		},
	}
	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.name), func() {
			suite.SetupTest()
			ctx = suite.network.GetContext()

			tc.malleate()

			err := suite.network.App.Erc20Keeper.OnTimeoutPacket(ctx, channeltypes.Packet{}, data)
			if tc.expPass {
				suite.Require().NoError(err)
				// check balance is the same as expected
				balance := suite.network.App.Erc20Keeper.BalanceOf(
					ctx, contracts.ERC20MinterBurnerDecimalsContract.ABI,
					pair.GetERC20Contract(),
					common.BytesToAddress(sender.Bytes()),
				)
				suite.Require().Equal(tc.expERC20.Int64(), balance.Int64())
			} else {
				tc.expErrorEvents()
			}
		})
	}
}
