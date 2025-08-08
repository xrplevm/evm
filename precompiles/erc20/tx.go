package erc20

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/cosmos/evm/utils"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

const (
	// TransferMethod defines the ABI method name for the ERC-20 transfer
	// transaction.
	TransferMethod = "transfer"
	// TransferFromMethod defines the ABI method name for the ERC-20 transferFrom
	// transaction.
	TransferFromMethod = "transferFrom"
	// MintMethod defines the ABI method name for the ERC-20 mint transaction.
	MintMethod = "mint"
	// BurnMethod defines the ABI method name for the ERC-20 burn transaction.
	BurnMethod = "burn"
	// Burn0Method defines the ABI method name for burn transaction with 2 arguments (spender, amount).
	Burn0Method = "burn0"
	// BurnFromMethod defines the ABI method name for the ERC-20 burnFrom transaction.
	BurnFromMethod = "burnFrom"
	// TransferOwnershipMethod defines the ABI method name for the ERC-20 transferOwnership transaction.
	TransferOwnershipMethod = "transferOwnership"
	// ApproveMethod defines the ABI method name for ERC-20 Approve
	// transaction.
	ApproveMethod = "approve"
	// DecreaseAllowanceMethod defines the ABI method name for the DecreaseAllowance
	// transaction.
	DecreaseAllowanceMethod = "decreaseAllowance"
	// IncreaseAllowanceMethod defines the ABI method name for the IncreaseAllowance
	// transaction.
	IncreaseAllowanceMethod = "increaseAllowance"
)

// ZeroAddress represents the zero address
var ZeroAddress = common.Address{}

// Transfer executes a direct transfer from the caller address to the
// destination address.
func (p *Precompile) Transfer(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	from := contract.Caller()
	to, amount, err := ParseTransferArgs(args)
	if err != nil {
		return nil, err
	}

	return p.transfer(ctx, contract, stateDB, method, from, to, amount)
}

// TransferFrom executes a transfer on behalf of the specified from address in
// the call data to the destination address.
func (p *Precompile) TransferFrom(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	from, to, amount, err := ParseTransferFromArgs(args)
	if err != nil {
		return nil, err
	}

	return p.transfer(ctx, contract, stateDB, method, from, to, amount)
}

// transfer is a common function that handles transfers for the ERC-20 Transfer
// and TransferFrom methods. It executes a bank Send message. If the spender isn't
// the sender of the transfer, it checks the allowance and updates it accordingly.
func (p *Precompile) transfer(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	from, to common.Address,
	amount *big.Int,
) (data []byte, err error) {
	coins := sdk.Coins{{Denom: p.tokenPair.Denom, Amount: math.NewIntFromBigInt(amount)}}

	msg := banktypes.NewMsgSend(from.Bytes(), to.Bytes(), coins)

	if err = msg.Amount.Validate(); err != nil {
		return nil, err
	}

	isTransferFrom := method.Name == TransferFromMethod
	spenderAddr := contract.Caller()
	newAllowance := big.NewInt(0)

	if isTransferFrom {
		spenderAddr := contract.Caller()

		prevAllowance, err := p.erc20Keeper.GetAllowance(ctx, p.Address(), from, spenderAddr)
		if err != nil {
			return nil, ConvertErrToERC20Error(err)
		}

		newAllowance := new(big.Int).Sub(prevAllowance, amount)
		if newAllowance.Sign() < 0 {
			return nil, ErrInsufficientAllowance
		}

		if newAllowance.Sign() == 0 {
			// If the new allowance is 0, we need to delete it from the store.
			err = p.erc20Keeper.DeleteAllowance(ctx, p.Address(), from, spenderAddr)
		} else {
			// If the new allowance is not 0, we need to set it in the store.
			err = p.erc20Keeper.SetAllowance(ctx, p.Address(), from, spenderAddr, newAllowance)
		}
		if err != nil {
			return nil, ConvertErrToERC20Error(err)
		}
	}

	msgSrv := NewMsgServerImpl(p.BankKeeper)
	if err = msgSrv.Send(ctx, msg); err != nil {
		// This should return an error to avoid the contract from being executed and an event being emitted
		return nil, ConvertErrToERC20Error(err)
	}

	// TODO: Properly handle native balance changes via the balance handler.
	// Currently, decimal conversion issues exist with the precisebank module.
	// As a temporary workaround, balances are adjusted directly using add/sub operations.
	evmDenom := evmtypes.GetEVMCoinDenom()
	if p.tokenPair.Denom == evmDenom {
		convertedAmount, err := utils.Uint256FromBigInt(evmtypes.ConvertAmountTo18DecimalsBigInt(amount))
		if err != nil {
			return nil, err
		}

		stateDB.SubBalance(from, convertedAmount, tracing.BalanceChangeUnspecified)
		stateDB.AddBalance(to, convertedAmount, tracing.BalanceChangeUnspecified)
	}

	if err = p.EmitTransferEvent(ctx, stateDB, from, to, amount); err != nil {
		return nil, err
	}

	// NOTE: if it's a direct transfer, we return here but if used through transferFrom,
	// we need to emit the approval event with the new allowance.
	if isTransferFrom {
		if err = p.EmitApprovalEvent(ctx, stateDB, from, spenderAddr, newAllowance); err != nil {
			return nil, err
		}
	}

	return method.Outputs.Pack(true)
}

// Mint executes a mint of the caller's tokens.
func (p *Precompile) Mint(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	to, amount, err := ParseMintArgs(args)
	if err != nil {
		return nil, err
	}

	minterAddr := contract.Caller()
	minter := sdk.AccAddress(minterAddr.Bytes())
	toAddr := sdk.AccAddress(to.Bytes())

	err = p.erc20Keeper.MintCoins(ctx, minter, toAddr, math.NewIntFromBigInt(amount), p.tokenPair.GetERC20Contract().Hex())
	if err != nil {
		return nil, ConvertErrToERC20Error(err)
	}

	// TODO: Properly handle native balance changes via the balance handler.
	// Currently, decimal conversion issues exist with the precisebank module.
	// As a temporary workaround, balances are adjusted directly using add/sub operations.
	evmDenom := evmtypes.GetEVMCoinDenom()
	if p.tokenPair.Denom == evmDenom {
		convertedAmount, err := utils.Uint256FromBigInt(evmtypes.ConvertAmountTo18DecimalsBigInt(amount))
		if err != nil {
			return nil, err
		}

		stateDB.AddBalance(to, convertedAmount, tracing.BalanceChangeUnspecified)
	}

	if err = p.EmitTransferEvent(ctx, stateDB, ZeroAddress, to, amount); err != nil {
		return nil, err
	}

	return method.Outputs.Pack()
}

// Burn executes a burn of the caller's tokens.
func (p *Precompile) Burn(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	amount, err := ParseBurnArgs(args)
	if err != nil {
		return nil, err
	}

	burnerAddr := contract.Caller()

	if err := p.burn(ctx, stateDB, burnerAddr, amount); err != nil {
		return nil, err
	}

	return method.Outputs.Pack()
}

// Burn0 executes a burn of the spender's tokens.
func (p *Precompile) Burn0(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	spender, amount, err := ParseBurn0Args(args)
	if err != nil {
		return nil, err
	}

	owner, err := sdk.AccAddressFromBech32(p.tokenPair.OwnerAddress)
	if err != nil {
		return nil, err
	}
	sender := sdk.AccAddress(contract.Caller().Bytes())

	if !sender.Equals(owner) {
		return nil, ConvertErrToERC20Error(ErrSenderIsNotOwner)
	}

	if err := p.burn(ctx, stateDB, spender, amount); err != nil {
		return nil, err
	}

	return method.Outputs.Pack()
}

// BurnFrom executes a burn of the caller's tokens.
func (p *Precompile) BurnFrom(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	owner, amount, err := ParseBurnFromArgs(args)
	if err != nil {
		return nil, err
	}

	coins := sdk.Coins{{Denom: p.tokenPair.Denom, Amount: math.NewIntFromBigInt(amount)}}

	msg := banktypes.NewMsgSend(owner.Bytes(), ZeroAddress.Bytes(), coins)

	if err = msg.Amount.Validate(); err != nil {
		return nil, err
	}

	isTransferFrom := method.Name == TransferFromMethod
	spenderAddr := contract.Caller()
	newAllowance := big.NewInt(0)

	if isTransferFrom {
		spenderAddr := contract.Caller()

		prevAllowance, err := p.erc20Keeper.GetAllowance(ctx, p.Address(), owner, spenderAddr)
		if err != nil {
			return nil, ConvertErrToERC20Error(err)
		}

		newAllowance := new(big.Int).Sub(prevAllowance, amount)
		if newAllowance.Sign() < 0 {
			return nil, ErrInsufficientAllowance
		}

		if newAllowance.Sign() == 0 {
			// If the new allowance is 0, we need to delete it from the store.
			err = p.erc20Keeper.DeleteAllowance(ctx, p.Address(), owner, spenderAddr)
		} else {
			// If the new allowance is not 0, we need to set it in the store.
			err = p.erc20Keeper.SetAllowance(ctx, p.Address(), owner, spenderAddr, newAllowance)
		}
		if err != nil {
			return nil, ConvertErrToERC20Error(err)
		}
	}

	msgSrv := NewMsgServerImpl(p.BankKeeper)
	if err = msgSrv.Send(ctx, msg); err != nil {
		// This should return an error to avoid the contract from being executed and an event being emitted
		return nil, ConvertErrToERC20Error(err)
	}

	// TODO: Properly handle native balance changes via the balance handler.
	// Currently, decimal conversion issues exist with the precisebank module.
	// As a temporary workaround, balances are adjusted directly using add/sub operations.
	evmDenom := evmtypes.GetEVMCoinDenom()
	if p.tokenPair.Denom == evmDenom {
		convertedAmount, err := utils.Uint256FromBigInt(evmtypes.ConvertAmountTo18DecimalsBigInt(amount))
		if err != nil {
			return nil, err
		}

		stateDB.SubBalance(owner, convertedAmount, tracing.BalanceChangeUnspecified)
	}

	if err = p.EmitTransferEvent(ctx, stateDB, owner, ZeroAddress, amount); err != nil {
		return nil, err
	}

	// NOTE: if it's a direct transfer, we return here but if used through transferFrom,
	// we need to emit the approval event with the new allowance.
	if isTransferFrom {
		if err = p.EmitApprovalEvent(ctx, stateDB, owner, spenderAddr, newAllowance); err != nil {
			return nil, err
		}
	}

	return method.Outputs.Pack()
}

// TransferOwnership executes a transfer of ownership of the token.
func (p *Precompile) TransferOwnership(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	newOwner, err := ParseTransferOwnershipArgs(args)
	if err != nil {
		return nil, err
	}

	sender := sdk.AccAddress(contract.Caller().Bytes())

	if p.tokenPair.OwnerAddress != sender.String() {
		return nil, ConvertErrToERC20Error(ErrSenderIsNotOwner)
	}

	err = p.erc20Keeper.TransferOwnership(ctx, sender, sdk.AccAddress(newOwner.Bytes()), p.tokenPair.GetERC20Contract().Hex())
	if err != nil {
		return nil, ConvertErrToERC20Error(err)
	}

	p.tokenPair.OwnerAddress = newOwner.String()

	if err = p.EmitTransferOwnershipEvent(ctx, stateDB, contract.Caller(), newOwner); err != nil {
		return nil, err
	}

	return method.Outputs.Pack()
}

// burn is a common function that handles burns for the ERC-20 Burn
// and BurnFrom methods. It executes a bank BurnCoins message.
func (p *Precompile) burn(ctx sdk.Context, stateDB vm.StateDB, burnerAddr common.Address, amount *big.Int) error {
	burner := sdk.AccAddress(burnerAddr.Bytes())

	err := p.erc20Keeper.BurnCoins(ctx, burner, math.NewIntFromBigInt(amount), p.tokenPair.GetERC20Contract().Hex())
	if err != nil {
		return ConvertErrToERC20Error(err)
	}

	// TODO: Properly handle native balance changes via the balance handler.
	// Currently, decimal conversion issues exist with the precisebank module.
	// As a temporary workaround, balances are adjusted directly using add/sub operations.
	evmDenom := evmtypes.GetEVMCoinDenom()
	if p.tokenPair.Denom == evmDenom {
		convertedAmount, err := utils.Uint256FromBigInt(evmtypes.ConvertAmountTo18DecimalsBigInt(amount))
		if err != nil {
			return err
		}

		stateDB.SubBalance(burnerAddr, convertedAmount, tracing.BalanceChangeUnspecified)
	}

	return p.EmitTransferEvent(ctx, stateDB, burnerAddr, ZeroAddress, amount)
}
