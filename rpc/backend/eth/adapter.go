package eth

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	legacytypes "github.com/cosmos/evm/rpc/types/legacy"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

// AdaptEthTxMsg converts an sdk.Msg to our unified TxMsg interface.
// This function tries to match the message to known MsgEthereumTx types
// and returns an appropriate adapter. This makes it easy to add support
// for new message types in the future by adding new case statements.
func AdaptEthTxMsg(msg sdk.Msg) (TxMsg, error) {
	switch m := msg.(type) {
	case *evmtypes.MsgEthereumTx:
		// New EVM type
		return &NewEVMEthTxMsg{MsgEthereumTx: m}, nil
	case *legacytypes.MsgEthereumTx:
		// Legacy Evmos type
		return &LegacyEvmosEthTxMsg{MsgEthereumTx: m}, nil
	default:
		return nil, fmt.Errorf("unsupported message type: %T", msg)
	}
}

// TryAdaptEthTxMsg attempts to adapt a message to TxMsg.
// Returns nil if the message is not an Ethereum transaction type.
func TryAdaptEthTxMsg(msg sdk.Msg) TxMsg {
	adapted, err := AdaptEthTxMsg(msg)
	if err != nil {
		return nil
	}
	return adapted
}
