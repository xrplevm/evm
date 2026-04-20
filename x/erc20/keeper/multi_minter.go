package keeper

import (
	"slices"
	"strings"

	"github.com/cosmos/evm/x/erc20/types"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AddMinterAddress adds a new minter address to the token pair.
func (k Keeper) AddMinterAddress(ctx sdk.Context, newMinter sdk.AccAddress, token string) error {
	pair, found := k.GetTokenPair(ctx, k.GetTokenPairID(ctx, token))
	newMinterStr := newMinter.String()
	if !found {
		return errorsmod.Wrapf(types.ErrTokenPairNotFound, "token '%s' not registered", token)
	}

	if !pair.IsNativeCoin() {
		return errorsmod.Wrapf(types.ErrExternalTokenNotSupported, "token '%s'", token)
	}

	if isAddressInOwnerAddresses(pair.OwnerAddresses, newMinterStr) {
		return errorsmod.Wrapf(types.ErrMinterAlreadyExists, "address '%s'", newMinterStr)
	}

	newAddresses := append(pair.OwnerAddresses, newMinterStr)
	k.SetTokenPairOwnerAddresses(ctx, pair, newAddresses)

	emitEventOwnerAddressesChange(ctx, types.TypeMsgAddMinter, token, newMinterStr, newAddresses)
	return nil
}

// RemoveMinterAddress removes a minter address from the token pair.
func (k Keeper) RemoveMinterAddress(ctx sdk.Context, delMinter sdk.AccAddress, token string) error {
	pair, found := k.GetTokenPair(ctx, k.GetTokenPairID(ctx, token))
	delMinterStr := delMinter.String()
	if !found {
		return errorsmod.Wrapf(types.ErrTokenPairNotFound, "token '%s' not registered", token)
	}

	if !isAddressInOwnerAddresses(pair.OwnerAddresses, delMinterStr) {
		return errorsmod.Wrapf(types.ErrMinterNotFound, "address '%s'", delMinterStr)
	}

	if len(pair.OwnerAddresses) == 1 {
		return errorsmod.Wrapf(types.ErrCannotRemoveLastMinter, "token '%s'", token)
	}

	newAddresses := slices.DeleteFunc(pair.OwnerAddresses, func(v string) bool {
		return v == delMinterStr
	})
	k.SetTokenPairOwnerAddresses(ctx, pair, newAddresses)

	emitEventOwnerAddressesChange(ctx, types.TypeMsgRemoveMinter, token, delMinterStr, newAddresses)
	return nil
}

func isAddressInOwnerAddresses(ownerAddresses []string, address string) bool {
	for _, addr := range ownerAddresses {
		if addr == address {
			return true
		}
	}
	return false
}

func emitEventOwnerAddressesChange(ctx sdk.Context, eventType string, token string, minterStr string, ownerAddresses []string) {
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyAction, eventType),
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
			sdk.NewAttribute(types.AttributeKeyToken, token),
			sdk.NewAttribute(types.AttributeKeyMinterAddress, minterStr),
			sdk.NewAttribute(types.AttributeKeyOwnerAddresses, strings.Join(ownerAddresses, ",")),
		),
	)
}

func (k Keeper) GetOwnerAddresses(ctx sdk.Context, contractAddress string) []string {
	pair, found := k.GetTokenPair(ctx, k.GetTokenPairID(ctx, contractAddress))
	if !found {
		return []string{}
	}

	return pair.OwnerAddresses
}

func (k Keeper) MigrateOwnerAddresses(ctx sdk.Context) {
	k.IterateTokenPairs(ctx, func(tokenPair types.TokenPair) (stop bool) {
		if tokenPair.ContractOwner == types.OWNER_MODULE && tokenPair.OwnerAddress != "" {
			tokenPair.OwnerAddresses = []string{tokenPair.OwnerAddress}
			tokenPair.OwnerAddress = ""
			k.SetTokenPair(ctx, tokenPair)
		}
		return false
	})
}
