package eth

import (
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

// TxMsg is a unified interface for different versions of MsgEthereumTx
// This interface abstracts the differences between the old Evmos format
// and the new EVM format, making it easy to add support for future versions.
type TxMsg interface {
	// AsTransaction returns the underlying Ethereum transaction
	AsTransaction() *ethtypes.Transaction
	// Hash returns the transaction hash
	Hash() common.Hash
	// GetSenderLegacy returns the sender address, recovering from signature if needed
	GetSenderLegacy(signer ethtypes.Signer) (common.Address, error)
	// GetGas returns the gas limit
	GetGas() uint64
}
