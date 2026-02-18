// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)
package types

import (
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
)

const maxBitLen = 256

// SafeNewIntFromBigInt constructs Int from big.Int, return error if more than 256bits
func SafeNewIntFromBigInt(i *big.Int) (sdkmath.Int, error) {
	if !IsValidInt256(i) {
		return sdkmath.NewInt(0), fmt.Errorf("big int out of bound: %s", i)
	}
	return sdkmath.NewIntFromBigInt(i), nil
}

// IsValidInt256 check the bound of 256 bit number
func IsValidInt256(i *big.Int) bool {
	return i == nil || i.BitLen() <= maxBitLen
}

// DeriveChainID derives the chain id from the given v parameter.
//
// CONTRACT: v value is either:
//
//   - {0,1} + CHAIN_ID * 2 + 35, if EIP155 is used
//   - {0,1} + 27, otherwise
//
// Ref: https://github.com/ethereum/EIPs/blob/master/EIPS/eip-155.md
func DeriveChainID(v *big.Int) *big.Int {
	if v == nil || v.Sign() < 1 {
		return nil
	}

	if v.BitLen() <= 64 {
		v := v.Uint64()
		if v == 27 || v == 28 {
			return new(big.Int)
		}

		if v < 35 {
			return nil
		}

		// V MUST be of the form {0,1} + CHAIN_ID * 2 + 35
		return new(big.Int).SetUint64((v - 35) / 2)
	}
	v = new(big.Int).Sub(v, big.NewInt(35))
	return v.Div(v, big.NewInt(2))
}

// RawSignatureValues is a helper function
// that parses the v,r and s fields of an Ethereum transaction
func RawSignatureValues(vBz, rBz, sBz []byte) (v, r, s *big.Int) {
	if len(vBz) > 0 {
		v = new(big.Int).SetBytes(vBz)
	}
	if len(rBz) > 0 {
		r = new(big.Int).SetBytes(rBz)
	}
	if len(sBz) > 0 {
		s = new(big.Int).SetBytes(sBz)
	}
	return v, r, s
}

// EffectiveGasPrice computes the effective gas price based on eip-1559 rules
// `effectiveGasPrice = min(baseFee + tipCap, feeCap)`
func EffectiveGasPrice(baseFee, feeCap, tipCap *big.Int) *big.Int {
	return bigMin(new(big.Int).Add(tipCap, baseFee), feeCap)
}

// bigMin returns the smaller of a and b
func bigMin(a, b *big.Int) *big.Int {
	if a.Cmp(b) < 0 {
		return a
	}
	return b
}
