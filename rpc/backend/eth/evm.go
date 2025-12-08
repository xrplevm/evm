package eth

import (
	"fmt"

	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

var _ TxMsg = NewEVMEthTxMsg{}

// NewEVMEthTxMsg creates an adapter for the new EVM MsgEthereumTx
type NewEVMEthTxMsg struct {
	*evmtypes.MsgEthereumTx
}

// AsTransaction returns the underlying Ethereum transaction for the new EVM type
func (m NewEVMEthTxMsg) AsTransaction() *ethtypes.Transaction {
	return m.MsgEthereumTx.AsTransaction()
}

// Hash returns the transaction hash for the new EVM type
func (m NewEVMEthTxMsg) Hash() common.Hash {
	return m.MsgEthereumTx.Hash()
}

// GetSenderLegacy returns the sender address for the new EVM type
func (m NewEVMEthTxMsg) GetSenderLegacy(signer ethtypes.Signer) (common.Address, error) {
	return m.MsgEthereumTx.GetSenderLegacy(signer)
}

// GetGas returns the gas limit for the new EVM type
func (m NewEVMEthTxMsg) GetGas() uint64 {
	return m.MsgEthereumTx.GetGas()
}

// ConvertToNewEVMMsg converts any TxMsg to the new EVM MsgEthereumTx format.
// This is useful when we need to work with code that specifically requires
// the new format. For legacy messages, this creates a new message with
// the converted data.
func ConvertToNewEVMMsg(msg TxMsg, signer ethtypes.Signer) (*evmtypes.MsgEthereumTx, error) {
	// If it's already the new format, return as-is
	if newMsg, ok := msg.(*NewEVMEthTxMsg); ok {
		return newMsg.MsgEthereumTx, nil
	}

	// For legacy messages, create a new message
	tx := msg.AsTransaction()
	sender, err := msg.GetSenderLegacy(signer)
	if err != nil {
		return nil, fmt.Errorf("failed to get sender: %w", err)
	}

	newMsg := &evmtypes.MsgEthereumTx{
		From: sender.Bytes(),
	}
	newMsg.FromEthereumTx(tx)

	return newMsg, nil
}
