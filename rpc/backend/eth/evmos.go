package eth

import (
	"math/big"

	legacytypes "github.com/cosmos/evm/rpc/types/legacy"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

var _ TxMsg = LegacyEvmosEthTxMsg{}

// LegacyEvmosEthTxMsg creates an adapter for the old Evmos MsgEthereumTx
type LegacyEvmosEthTxMsg struct {
	*legacytypes.MsgEthereumTx
}

// AsTransaction returns the underlying Ethereum transaction for the legacy Evmos type
func (m LegacyEvmosEthTxMsg) AsTransaction() *ethtypes.Transaction {
	return m.MsgEthereumTx.AsTransaction()
}

// Hash returns the transaction hash for the legacy Evmos type
func (m LegacyEvmosEthTxMsg) Hash() common.Hash {
	// The old type stores hash as hex string, convert it back
	if m.MsgEthereumTx.Hash != "" {
		return common.HexToHash(m.MsgEthereumTx.Hash)
	}
	// Fallback: compute from transaction
	return m.AsTransaction().Hash()
}

// GetSenderLegacy returns the sender address for the legacy Evmos type
func (m LegacyEvmosEthTxMsg) GetSenderLegacy(signer ethtypes.Signer) (common.Address, error) {
	// If From field is set, use it
	if m.MsgEthereumTx.From != "" {
		return common.HexToAddress(m.MsgEthereumTx.From), nil
	}
	// Otherwise recover from signature
	tx := m.AsTransaction()
	sender, err := signer.Sender(tx)
	if err != nil {
		return common.Address{}, err
	}
	// Cache the sender for future use
	m.MsgEthereumTx.From = sender.Hex()
	return sender, nil
}

// GetGas returns the gas limit for the legacy Evmos type
func (m LegacyEvmosEthTxMsg) GetGas() uint64 {
	return m.MsgEthereumTx.GetGas()
}

// ConvertToNewEVMMsgWithChainID is like ConvertToNewEVMMsg but uses chainID to create signer
func ConvertToNewEVMMsgWithChainID(msg TxMsg, chainID *big.Int) (*evmtypes.MsgEthereumTx, error) {
	// If it's already the new format, return as-is
	if newMsg, ok := msg.(*NewEVMEthTxMsg); ok {
		return newMsg.MsgEthereumTx, nil
	}

	tx := msg.AsTransaction()
	var signer ethtypes.Signer
	if tx.Protected() {
		signer = ethtypes.LatestSignerForChainID(tx.ChainId())
	} else {
		signer = ethtypes.FrontierSigner{}
	}

	return ConvertToNewEVMMsg(msg, signer)
}
