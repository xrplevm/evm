package erc20

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// revive:disable-next-line exported
type Erc20Keeper interface {
	GetAllowance(ctx sdk.Context, erc20 common.Address, owner common.Address, spender common.Address) (*big.Int, error)
	SetAllowance(ctx sdk.Context, erc20 common.Address, owner common.Address, spender common.Address, value *big.Int) error
	DeleteAllowance(ctx sdk.Context, erc20 common.Address, owner common.Address, spender common.Address) error
	MintCoins(ctx sdk.Context, sender, to sdk.AccAddress, amount math.Int, token string) error
	BurnCoins(ctx sdk.Context, sender sdk.AccAddress, amount math.Int, token string) error
	GetTokenPairOwnerAddress(ctx sdk.Context, token string) (sdk.AccAddress, error)
	TransferOwnership(ctx sdk.Context, sender sdk.AccAddress, newOwner sdk.AccAddress, token string) error
}
