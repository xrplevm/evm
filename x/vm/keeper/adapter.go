package keeper

import (
	"fmt"
	"strconv"
	"strings"

	legacytypes "github.com/cosmos/evm/rpc/types/legacy"
	"github.com/cosmos/evm/x/vm/types"

	"github.com/cosmos/cosmos-sdk/codec"
)

const legacyExtraEIPPrefix = "ethereum_"

// AdaptUnmarshalParams unmarshals EVM params bytes, handling both the current
// cosmos.evm schema and the legacy Ethermint schema transparently.
//
// The two schemas are wire-incompatible: several field numbers changed meaning
// between Ethermint and cosmos.evm (e.g. field 8 was EVMChannels, now it is
// AccessControl), and ExtraEIPs changed from repeated bytes to packed varint.
// Because of this, we attempt the current schema first, then fall back to the
// legacy schema and convert field by field.
//
// The fallback relies on proto field 10 having incompatible wire types between
// the two schemas: legacy uses repeated string (wire type 2) for
// ActiveStaticPrecompiles, while cosmos.evm uses uint64 (wire type 0) for
// HistoryServeWindow. When legacy bytes contain ActiveStaticPrecompiles — which
// is always the case in production — Unmarshal returns a wire type error,
// triggering the legacy fallback path.
//
// Fields that only exist in the new schema (HistoryServeWindow,
// ExtendedDenomOptions) are left at their zero values when converting from
// legacy params.
func AdaptUnmarshalParams(cdc codec.BinaryCodec, bz []byte) (types.Params, error) {
	var params types.Params
	if err := cdc.Unmarshal(bz, &params); err == nil {
		return params, nil
	}

	// Current schema failed — attempt the legacy Ethermint layout.
	var legacy legacytypes.Params
	if err := cdc.Unmarshal(bz, &legacy); err != nil {
		return types.Params{}, fmt.Errorf("unmarshal EVM params: neither current nor legacy schema matched: %w", err)
	}

	// Convert ExtraEIPs from legacy string format ("ethereum_3855") to int64.
	// Same logic as the v9 upgrade migration in app/upgrades/v9/upgrades.go.
	eips := make([]int64, len(legacy.ExtraEIPs))
	for i, extraEIP := range legacy.ExtraEIPs {
		sanitized := strings.TrimPrefix(extraEIP, legacyExtraEIPPrefix)
		intEIP, err := strconv.ParseInt(sanitized, 10, 64)
		if err != nil {
			return types.Params{}, fmt.Errorf("invalid legacy ExtraEIP %q: %w", extraEIP, err)
		}
		eips[i] = intEIP
	}

	// Convert AccessControl — identical structure, different Go types.
	accessControl := types.AccessControl{
		Create: types.AccessControlType{
			AccessType:        types.AccessType(legacy.AccessControl.Create.AccessType),
			AccessControlList: legacy.AccessControl.Create.AccessControlList,
		},
		Call: types.AccessControlType{
			AccessType:        types.AccessType(legacy.AccessControl.Call.AccessType),
			AccessControlList: legacy.AccessControl.Call.AccessControlList,
		},
	}

	parsedParams := types.Params{
		EvmDenom:                legacy.EvmDenom,
		ExtraEIPs:               eips,
		EVMChannels:             legacy.EVMChannels,
		AccessControl:           accessControl,
		ActiveStaticPrecompiles: legacy.ActiveStaticPrecompiles,
		// HistoryServeWindow and ExtendedDenomOptions are new fields that don't
		// exist in legacy — Go zero values (0 and nil) are correct defaults.
	}

	return parsedParams, nil
}
