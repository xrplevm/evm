package types

import (
	legacytypes "github.com/cosmos/evm/rpc/types/legacy"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// RegisterInterfaces registers the CometBFT concrete client-related
// implementations and interfaces.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdktypes.AccountI)(nil),
		// TODO: uncomment after moving into migrations for EVM version
		// &EthAccount{},
	)
	registry.RegisterImplementations(
		(*authtypes.GenesisAccount)(nil),
		// TODO: uncomment after moving into migrations for EVM version
		// &EthAccount{},
	)
	registry.RegisterImplementations(
		(*tx.TxExtensionOptionI)(nil),
		&ExtensionOptionsWeb3Tx{},
		&ExtensionOptionDynamicFeeTx{},
	)

	// Register the TxData interface for legacy ethermint transaction types.
	// These are needed to decode pre-v9 upgrade transactions that use ethermint.evm.v1 proto package.
	// The proto types are registered in evm/rpc/types/legacy/tx.pb.go via init().
	registry.RegisterInterface(
		"ethermint.evm.v1.TxData",
		(*legacytypes.TxData)(nil),
		&legacytypes.LegacyTx{},
		&legacytypes.AccessListTx{},
		&legacytypes.DynamicFeeTx{},
	)

	registry.RegisterImplementations(
		(*sdktypes.Msg)(nil),
		&legacytypes.MsgEthereumTx{},
		&legacytypes.MsgUpdateParams{},
	)

	registry.RegisterImplementations(
		(*tx.TxExtensionOptionI)(nil),
		&legacytypes.ExtensionOptionsEthereumTx{},
	)
}
