package types

import (
	errorsmod "cosmossdk.io/errors"
	"github.com/ethereum/go-ethereum/common"

	"github.com/cometbft/cometbft/crypto/tmhash"

	"github.com/cosmos/evm/utils"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NewTokenPairSTRv2 creates a new TokenPair instance in the context of the
// Single Token Representation v2.
//
// It derives the ERC-20 address from the hex suffix of the IBC denomination
// (e.g. ibc/DF63978F803A2E27CA5CC9B7631654CCF0BBC788B3B7F0A10200508E37C70992).
func NewTokenPairSTRv2(denom string) (TokenPair, error) {
	address, err := utils.GetIBCDenomAddress(denom)
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{
		Erc20Address:  address.String(),
		Denom:         denom,
		Enabled:       true,
		ContractOwner: OWNER_MODULE,
	}, nil
}

// NewTokenPair returns an instance of TokenPair
func NewTokenPair(erc20Address common.Address, denom string, contractOwner Owner) TokenPair {
	return TokenPair{
		Erc20Address:  erc20Address.String(),
		Denom:         denom,
		Enabled:       true,
		ContractOwner: contractOwner,
	}
}

// GetID returns the SHA256 hash of the ERC20 address and denomination
func (tp TokenPair) GetID() []byte {
	id := tp.Erc20Address + "|" + tp.Denom
	return tmhash.Sum([]byte(id))
}

// GetErc20Contract casts the hex string address of the ERC20 to common.Address
func (tp TokenPair) GetERC20Contract() common.Address {
	return common.HexToAddress(tp.Erc20Address)
}

// SetOwnerAddresses sets the authorized minter addresses for the token pair
func (tp *TokenPair) SetOwnerAddresses(addresses []string) {
	tp.OwnerAddresses = addresses
}

// Validate performs a stateless validation of a TokenPair
func (tp TokenPair) Validate() error {
	if err := sdk.ValidateDenom(tp.Denom); err != nil {
		return err
	}

	if err := utils.ValidateAddress(tp.Erc20Address); err != nil {
		return err
	}

	if tp.IsNativeCoin() {
		return validateOwnerAddresses(tp.OwnerAddresses)
	}

	// Externally owned tokens must not have owner addresses
	if len(tp.OwnerAddresses) > 0 {
		return errorsmod.Wrap(ErrExternalTokenNotSupported, "owner_addresses must be empty for externally owned tokens")
	}

	return nil
}

func validateOwnerAddresses(addresses []string) error {
	if len(addresses) == 0 {
		return errorsmod.Wrap(ErrInvalidOwnerAddresses, "owner addresses cannot be empty")
	}

	for _, addr := range addresses {
		if _, err := sdk.AccAddressFromBech32(addr); err != nil {
			return errorsmod.Wrapf(ErrInvalidOwnerAddresses, "invalid owner address: %s", addr)
		}
	}

	if utils.HasDuplicates(addresses) {
		return errorsmod.Wrap(ErrInvalidOwnerAddresses, "owner addresses cannot contain duplicates")
	}

	return nil
}

// IsAuthorizedMinter returns true if the given address is in the token pair's authorized minter set.
func (tp TokenPair) IsAuthorizedMinter(a sdk.AccAddress) bool {
	for _, addr := range tp.OwnerAddresses {
		ownerAddr, err := sdk.AccAddressFromBech32(addr)
		if err != nil {
			continue
		}

		if a.Equals(ownerAddr) {
			return true
		}
	}
	return false
}

// IsNativeCoin returns true if the owner of the ERC20 contract is the
// erc20 module account
func (tp TokenPair) IsNativeCoin() bool {
	return tp.ContractOwner == OWNER_MODULE
}

// IsNativeERC20 returns true if the owner of the ERC20 contract is an EOA.
func (tp TokenPair) IsNativeERC20() bool {
	return tp.ContractOwner == OWNER_EXTERNAL
}
