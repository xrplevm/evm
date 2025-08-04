package types

import (
	errorsmod "cosmossdk.io/errors"
)

// errors
var (
	ErrERC20Disabled            = errorsmod.Register(ModuleName, 2, "erc20 module is disabled")
	ErrInternalTokenPair        = errorsmod.Register(ModuleName, 3, "internal ethereum token mapping error")
	ErrTokenPairNotFound        = errorsmod.Register(ModuleName, 4, "token pair not found")
	ErrTokenPairAlreadyExists   = errorsmod.Register(ModuleName, 5, "token pair already exists")
	ErrUndefinedOwner           = errorsmod.Register(ModuleName, 6, "undefined owner of contract pair")
	ErrBalanceInvariance        = errorsmod.Register(ModuleName, 7, "post transfer balance invariant failed")
	ErrUnexpectedEvent          = errorsmod.Register(ModuleName, 8, "unexpected event")
	ErrABIPack                  = errorsmod.Register(ModuleName, 9, "contract ABI pack failed")
	ErrABIUnpack                = errorsmod.Register(ModuleName, 10, "contract ABI unpack failed")
	ErrEVMDenom                 = errorsmod.Register(ModuleName, 11, "EVM denomination registration")
	ErrEVMCall                  = errorsmod.Register(ModuleName, 12, "EVM call unexpected error")
	ErrERC20TokenPairDisabled   = errorsmod.Register(ModuleName, 13, "erc20 token pair is disabled")
	ErrInvalidIBC               = errorsmod.Register(ModuleName, 14, "invalid IBC transaction")
	ErrTokenPairOwnedByModule   = errorsmod.Register(ModuleName, 15, "token pair owned by module")
	ErrNativeConversionDisabled = errorsmod.Register(ModuleName, 16, "native coins manual conversion is disabled")
	ErrAllowanceNotFound        = errorsmod.Register(ModuleName, 17, "allowance not found")
	ErrInvalidAllowance         = errorsmod.Register(ModuleName, 18, "invalid allowance")
	ErrNegativeToken            = errorsmod.Register(ModuleName, 19, "token amount is negative")
	ErrExpectedEvent            = errorsmod.Register(ModuleName, 20, "expected event")
	ErrMinterIsNotOwner                   = errorsmod.Register(ModuleName, 21, "minter is not the owner")
	ErrSenderIsNotOwner                   = errorsmod.Register(ModuleName, 22, "sender is not the owner")
	ErrNonNativeCoinMintingDisabled       = errorsmod.Register(ModuleName, 23, "minting non-native coins is disabled")
	ErrNonNativeCoinBurningDisabled       = errorsmod.Register(ModuleName, 24, "burning non-native coins is disabled")
	ErrNonNativeTransferOwnershipDisabled = errorsmod.Register(ModuleName, 21, "transferring ownership of non-native coins is disabled")
)
