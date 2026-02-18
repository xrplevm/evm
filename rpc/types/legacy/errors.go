// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)
package types

import (
	errorsmod "cosmossdk.io/errors"
)

const (
	// ModuleName is the name of the legacy types module (for error registration)

	codeErrInvalidAmount = uint32(iota) + 2
	codeErrInvalidGasPrice
	codeErrInvalidGasFee
	codeErrInvalidGasCap
	codeErrGasOverflow
	codeErrInvalidGasLimit
)

var (
	// ErrInvalidAmount returns an error if a tx contains an invalid amount.
	ErrInvalidAmount = errorsmod.Register(ModuleName, codeErrInvalidAmount, "invalid transaction amount")

	// ErrInvalidGasPrice returns an error if an invalid gas price is provided to the tx.
	ErrInvalidGasPrice = errorsmod.Register(ModuleName, codeErrInvalidGasPrice, "invalid gas price")

	// ErrInvalidGasFee returns an error if the tx gas fee is out of bound.
	ErrInvalidGasFee = errorsmod.Register(ModuleName, codeErrInvalidGasFee, "invalid gas fee")

	// ErrInvalidGasCap returns an error if a the gas cap value is negative or invalid
	ErrInvalidGasCap = errorsmod.Register(ModuleName, codeErrInvalidGasCap, "invalid gas cap")

	// ErrGasOverflow returns an error if gas computation overflow/underflow
	ErrGasOverflow = errorsmod.Register(ModuleName, codeErrGasOverflow, "gas computation overflow/underflow")

	// ErrInvalidGasLimit returns an error if gas limit value is invalid
	ErrInvalidGasLimit = errorsmod.Register(ModuleName, codeErrInvalidGasLimit, "invalid gas limit")
)
