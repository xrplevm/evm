// SPDX-License-Identifier:LGPL-3.0-only

package keeper

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/cosmos/evm/x/erc20/types"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// RegisterERC20 creates a Cosmos coin and registers the token pair between the
// coin and the ERC20
func (k Keeper) registerERC20(
	ctx sdk.Context,
	contract common.Address,
) (*types.TokenPair, error) {
	// Check if ERC20 is already registered
	if k.IsERC20Registered(ctx, contract) {
		return nil, errorsmod.Wrapf(
			types.ErrTokenPairAlreadyExists, "token ERC20 contract already registered: %s", contract.String(),
		)
	}

	metadata, err := k.CreateCoinMetadata(ctx, contract)
	if err != nil {
		return nil, errorsmod.Wrap(
			err, "failed to create wrapped coin denom metadata for ERC20",
		)
	}

	pair := types.NewTokenPair(contract, metadata.Name, types.OWNER_EXTERNAL)
	err = k.SetToken(ctx, pair)
	if err != nil {
		return nil, err
	}
	return &pair, nil
}

// CreateCoinMetadata generates the metadata to represent the ERC20 token on
// evmos.
func (k Keeper) CreateCoinMetadata(
	ctx sdk.Context,
	contract common.Address,
) (*banktypes.Metadata, error) {
	strContract := contract.String()

	erc20Data, err := k.QueryERC20(ctx, contract)
	if err != nil {
		return nil, err
	}

	// Check if metadata already exists
	_, found := k.bankKeeper.GetDenomMetaData(ctx, types.CreateDenom(strContract))
	if found {
		return nil, errorsmod.Wrap(
			types.ErrInternalTokenPair, "denom metadata already registered",
		)
	}

	if k.IsDenomRegistered(ctx, types.CreateDenom(strContract)) {
		return nil, errorsmod.Wrapf(
			types.ErrInternalTokenPair, "coin denomination already registered: %s", erc20Data.Name,
		)
	}

	// base denomination
	base := types.CreateDenom(strContract)

	// create a bank denom metadata based on the ERC20 token ABI details
	// metadata name is should always be the contract since it's the key
	// to the bank store
	metadata := banktypes.Metadata{
		Description: types.CreateDenomDescription(strContract),
		Base:        base,
		// NOTE: Denom units MUST be increasing
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    base,
				Exponent: 0,
			},
		},
		Name:    types.CreateDenom(strContract),
		Symbol:  erc20Data.Symbol,
		Display: base,
	}

	// only append metadata if decimals > 0, otherwise validation fails
	if erc20Data.Decimals > 0 {
		nameSanitized := types.SanitizeERC20Name(erc20Data.Name)
		metadata.DenomUnits = append(
			metadata.DenomUnits,
			&banktypes.DenomUnit{
				Denom:    nameSanitized,
				Exponent: uint32(erc20Data.Decimals), //#nosec G115 -- int overflow is not a concern here
			},
		)
		metadata.Display = nameSanitized
	}

	if err := metadata.Validate(); err != nil {
		return nil, errorsmod.Wrapf(
			err, "ERC20 token data is invalid for contract %s", strContract,
		)
	}

	k.bankKeeper.SetDenomMetaData(ctx, metadata)

	return &metadata, nil
}

// ToggleConversion toggles conversion for a given token pair
func (k Keeper) toggleConversion(
	ctx sdk.Context,
	token string,
) (types.TokenPair, error) {
	id := k.GetTokenPairID(ctx, token)
	if len(id) == 0 {
		return types.TokenPair{}, errorsmod.Wrapf(
			types.ErrTokenPairNotFound, "token '%s' not registered by id", token,
		)
	}

	pair, found := k.GetTokenPair(ctx, id)
	if !found {
		return types.TokenPair{}, errorsmod.Wrapf(
			types.ErrTokenPairNotFound, "token '%s' not registered", token,
		)
	}

	pair.Enabled = !pair.Enabled
	k.SetTokenPair(ctx, pair)
	return pair, nil
}
