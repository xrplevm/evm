package keeper

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/cosmos/evm/utils"
	"github.com/cosmos/evm/x/erc20/types"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// CreateNewTokenPair creates a new token pair and stores it in the state.
func (k *Keeper) CreateNewTokenPair(ctx sdk.Context, denom string) (types.TokenPair, error) {
	pair, err := types.NewTokenPairSTRv2(denom)
	if err != nil {
		return types.TokenPair{}, err
	}
	if account := k.evmKeeper.GetAccount(ctx, pair.GetERC20Contract()); account != nil && account.IsContract() {
		return types.TokenPair{}, errorsmod.Wrapf(types.ErrTokenPairAlreadyExists, "token already exists for token %s", pair.Erc20Address)
	}
	err = k.SetToken(ctx, pair)
	if err != nil {
		return types.TokenPair{}, err
	}
	return pair, nil
}

// SetToken stores a token pair, denom map and erc20 map.
func (k *Keeper) SetToken(ctx sdk.Context, pair types.TokenPair) error {
	if k.IsDenomRegistered(ctx, pair.Denom) {
		return errorsmod.Wrapf(types.ErrTokenPairAlreadyExists, "token already exists for denom %s", pair.Denom)
	}
	if k.IsERC20Registered(ctx, pair.GetERC20Contract()) {
		return errorsmod.Wrapf(types.ErrTokenPairAlreadyExists, "token already exists for token %s", pair.Erc20Address)
	}
	k.SetTokenPair(ctx, pair)
	k.SetDenomMap(ctx, pair.Denom, pair.GetID())
	k.SetERC20Map(ctx, pair.GetERC20Contract(), pair.GetID())
	return nil
}

// GetTokenPairs gets all registered token tokenPairs.
func (k Keeper) GetTokenPairs(ctx sdk.Context) []types.TokenPair {
	tokenPairs := []types.TokenPair{}

	k.IterateTokenPairs(ctx, func(tokenPair types.TokenPair) (stop bool) {
		tokenPairs = append(tokenPairs, tokenPair)
		return false
	})

	return tokenPairs
}

// IterateTokenPairs iterates over all the stored token pairs.
func (k Keeper) IterateTokenPairs(ctx sdk.Context, cb func(tokenPair types.TokenPair) (stop bool)) {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, types.KeyPrefixTokenPair)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var tokenPair types.TokenPair
		k.cdc.MustUnmarshal(iterator.Value(), &tokenPair)

		if cb(tokenPair) {
			break
		}
	}
}

// GetTokenPairID returns the pair id for the specified token. Hex address or Denom can be used as token argument.
// If the token is not registered empty bytes are returned.
func (k Keeper) GetTokenPairID(ctx sdk.Context, token string) []byte {
	if common.IsHexAddress(token) {
		addr := common.HexToAddress(token)
		return k.GetERC20Map(ctx, addr)
	}
	return k.GetDenomMap(ctx, token)
}

// GetTokenPair gets a registered token pair from the identifier.
func (k Keeper) GetTokenPair(ctx sdk.Context, id []byte) (types.TokenPair, bool) {
	if id == nil {
		return types.TokenPair{}, false
	}

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPair)
	var tokenPair types.TokenPair
	bz := store.Get(id)
	if len(bz) == 0 {
		return types.TokenPair{}, false
	}

	k.cdc.MustUnmarshal(bz, &tokenPair)
	return tokenPair, true
}

// SetTokenPair stores a token pair.
func (k Keeper) SetTokenPair(ctx sdk.Context, tokenPair types.TokenPair) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPair)
	key := tokenPair.GetID()
	bz := k.cdc.MustMarshal(&tokenPair)
	store.Set(key, bz)
}

// DeleteTokenPair removes a token pair.
func (k Keeper) DeleteTokenPair(ctx sdk.Context, tokenPair types.TokenPair) {
	id := tokenPair.GetID()
	k.deleteTokenPair(ctx, id)
	k.deleteERC20Map(ctx, tokenPair.GetERC20Contract())
	k.deleteDenomMap(ctx, tokenPair.Denom)
	k.deleteAllowances(ctx, tokenPair.GetERC20Contract())
}

// deleteTokenPair deletes the token pair for the given id.
func (k Keeper) deleteTokenPair(ctx sdk.Context, id []byte) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPair)
	store.Delete(id)
}

// GetERC20Map returns the token pair id for the given address.
func (k Keeper) GetERC20Map(ctx sdk.Context, erc20 common.Address) []byte {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPairByERC20)
	return store.Get(erc20.Bytes())
}

// GetDenomMap returns the token pair id for the given denomination.
func (k Keeper) GetDenomMap(ctx sdk.Context, denom string) []byte {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPairByDenom)
	return store.Get([]byte(denom))
}

// SetERC20Map sets the token pair id for the given address.
func (k Keeper) SetERC20Map(ctx sdk.Context, erc20 common.Address, id []byte) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPairByERC20)
	store.Set(erc20.Bytes(), id)
}

// deleteERC20Map deletes the token pair id for the given address.
func (k Keeper) deleteERC20Map(ctx sdk.Context, erc20 common.Address) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPairByERC20)
	store.Delete(erc20.Bytes())
}

// SetDenomMap sets the token pair id for the denomination.
func (k Keeper) SetDenomMap(ctx sdk.Context, denom string, id []byte) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPairByDenom)
	store.Set([]byte(denom), id)
}

// deleteDenomMap deletes the token pair id for the given denom.
func (k Keeper) deleteDenomMap(ctx sdk.Context, denom string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPairByDenom)
	store.Delete([]byte(denom))
}

// IsTokenPairRegistered - check if registered token tokenPair is registered.
func (k Keeper) IsTokenPairRegistered(ctx sdk.Context, id []byte) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPair)
	return store.Has(id)
}

// IsERC20Registered check if registered ERC20 token is registered.
func (k Keeper) IsERC20Registered(ctx sdk.Context, erc20 common.Address) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPairByERC20)
	return store.Has(erc20.Bytes())
}

// IsDenomRegistered check if registered coin denom is registered.
func (k Keeper) IsDenomRegistered(ctx sdk.Context, denom string) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPairByDenom)
	return store.Has([]byte(denom))
}

// GetCoinAddress returns the corresponding ERC-20 contract address for the
// given denom.
// If the denom is not registered and its an IBC voucher, it returns the address
// from the hash of the ICS20's Denom Path.
func (k Keeper) GetCoinAddress(ctx sdk.Context, denom string) (common.Address, error) {
	id := k.GetDenomMap(ctx, denom)
	if len(id) == 0 {
		// if the denom is not registered, check if it is an IBC voucher
		return utils.GetIBCDenomAddress(denom)
	}

	tokenPair, found := k.GetTokenPair(ctx, id)
	if !found {
		// safety check, should never happen
		return common.Address{}, errorsmod.Wrapf(
			types.ErrTokenPairNotFound, "coin '%s' not registered", denom,
		)
	}

	return tokenPair.GetERC20Contract(), nil
}

// GetTokenDenom returns the denom associated with the tokenAddress or an error
// if the TokenPair does not exist.
func (k Keeper) GetTokenDenom(ctx sdk.Context, tokenAddress common.Address) (string, error) {
	tokenPairID := k.GetERC20Map(ctx, tokenAddress)
	if len(tokenPairID) == 0 {
		return "", errorsmod.Wrapf(
			types.ErrTokenPairNotFound, "token '%s' not registered", tokenAddress,
		)
	}

	tokenPair, found := k.GetTokenPair(ctx, tokenPairID)
	if !found {
		// safety check, should never happen
		return "", errorsmod.Wrapf(
			types.ErrTokenPairNotFound, "token '%s' not registered", tokenAddress,
		)
	}

	return tokenPair.Denom, nil
}
