package werc20

import (
	"math/big"

	"cosmossdk.io/math"
	"github.com/ethereum/go-ethereum/common"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Erc20Keeper interface {
	GetAllowance(ctx sdk.Context, erc20 common.Address, owner common.Address, spender common.Address) (*big.Int, error)
	SetAllowance(ctx sdk.Context, erc20 common.Address, owner common.Address, spender common.Address, value *big.Int) error
	DeleteAllowance(ctx sdk.Context, erc20 common.Address, owner common.Address, spender common.Address) error
	BurnCoins(ctx sdk.Context, sender sdk.AccAddress, amount math.Int, token string) error
	GetTokenPairOwnerAddress(ctx sdk.Context, token string) (sdk.AccAddress, error)
	TransferOwnership(ctx sdk.Context, sender sdk.AccAddress, newOwner sdk.AccAddress, token string) error
	MintCoins(ctx sdk.Context, sender, to sdk.AccAddress, amount math.Int, token string) error
}
