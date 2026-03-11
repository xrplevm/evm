package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	legacytypes "github.com/cosmos/evm/rpc/types/legacy"
	vmkeeper "github.com/cosmos/evm/x/vm/keeper"
	vmtypes "github.com/cosmos/evm/x/vm/types"

	"github.com/cosmos/cosmos-sdk/codec"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
)

func makeCodec() codec.BinaryCodec {
	encCfg := moduletestutil.MakeTestEncodingConfig()
	return encCfg.Codec
}

func TestAdaptUnmarshalParamsCurrentSchema(t *testing.T) {
	cdc := makeCodec()

	original := vmtypes.Params{
		EvmDenom:                "aXRP",
		ExtraEIPs:               []int64{3855, 3860},
		EVMChannels:             []string{"channel-0"},
		ActiveStaticPrecompiles: []string{"0x0000000000000000000000000000000000000800"},
		AccessControl: vmtypes.AccessControl{
			Create: vmtypes.AccessControlType{
				AccessType: vmtypes.AccessTypePermissionless,
			},
			Call: vmtypes.AccessControlType{
				AccessType: vmtypes.AccessTypePermissionless,
			},
		},
		HistoryServeWindow:   8192,
		ExtendedDenomOptions: &vmtypes.ExtendedDenomOptions{ExtendedDenom: "aXRP"},
	}

	bz, err := cdc.Marshal(&original)
	require.NoError(t, err)

	result, err := vmkeeper.AdaptUnmarshalParams(cdc, bz)
	require.NoError(t, err)

	require.Equal(t, original.EvmDenom, result.EvmDenom)
	require.Equal(t, original.ExtraEIPs, result.ExtraEIPs)
	require.Equal(t, original.EVMChannels, result.EVMChannels)
	require.Equal(t, original.ActiveStaticPrecompiles, result.ActiveStaticPrecompiles)
	require.Equal(t, original.HistoryServeWindow, result.HistoryServeWindow)
	require.Equal(t, original.ExtendedDenomOptions.ExtendedDenom, result.ExtendedDenomOptions.ExtendedDenom)
}

func TestAdaptUnmarshalParamsLegacySchema(t *testing.T) {
	cdc := makeCodec()

	legacy := legacytypes.Params{
		EvmDenom:  "aXRP",
		ExtraEIPs: []string{"ethereum_3855", "ethereum_3860"},
		ChainConfig: legacytypes.ChainConfig{
			DAOForkSupport: true,
		},
		AllowUnprotectedTxs:     false,
		EVMChannels:             []string{"channel-0", "channel-1"},
		ActiveStaticPrecompiles: []string{"0x0000000000000000000000000000000000000800"},
		AccessControl: legacytypes.AccessControl{
			Create: legacytypes.AccessControlType{
				AccessType:        legacytypes.AccessTypePermissioned,
				AccessControlList: []string{"0x1234567890abcdef1234567890abcdef12345678"},
			},
			Call: legacytypes.AccessControlType{
				AccessType: legacytypes.AccessTypePermissionless,
			},
		},
	}

	// Marshal with the legacy schema — these are the bytes that would be
	// stored in the KV store before the upgrade.
	bz, err := cdc.Marshal(&legacy)
	require.NoError(t, err)

	// AdaptUnmarshalParams should fail the current schema and fall back to legacy.
	result, err := vmkeeper.AdaptUnmarshalParams(cdc, bz)
	require.NoError(t, err)

	// Fields that exist in both schemas
	require.Equal(t, "aXRP", result.EvmDenom)
	require.Equal(t, []int64{3855, 3860}, result.ExtraEIPs)
	require.Equal(t, []string{"channel-0", "channel-1"}, result.EVMChannels)
	require.Equal(t, []string{"0x0000000000000000000000000000000000000800"}, result.ActiveStaticPrecompiles)

	// AccessControl should be converted
	require.Equal(t, vmtypes.AccessTypePermissioned, result.AccessControl.Create.AccessType)
	require.Equal(t, []string{"0x1234567890abcdef1234567890abcdef12345678"}, result.AccessControl.Create.AccessControlList)
	require.Equal(t, vmtypes.AccessTypePermissionless, result.AccessControl.Call.AccessType)

	// New fields should be zero values
	require.Equal(t, uint64(0), result.HistoryServeWindow)
	require.Nil(t, result.ExtendedDenomOptions)
}

func TestAdaptUnmarshalParamsLegacyMinimalParams(t *testing.T) {
	cdc := makeCodec()

	// Minimal legacy params — EvmDenom + ActiveStaticPrecompiles (field 10)
	// which is always present in production and triggers the wire type conflict.
	legacy := legacytypes.Params{
		EvmDenom:                "aXRP",
		ActiveStaticPrecompiles: []string{"0x0000000000000000000000000000000000000800"},
	}

	bz, err := cdc.Marshal(&legacy)
	require.NoError(t, err)

	result, err := vmkeeper.AdaptUnmarshalParams(cdc, bz)
	require.NoError(t, err)

	require.Equal(t, "aXRP", result.EvmDenom)
	require.Empty(t, result.ExtraEIPs)
	require.Empty(t, result.EVMChannels)
	require.Equal(t, []string{"0x0000000000000000000000000000000000000800"}, result.ActiveStaticPrecompiles)
}

func TestAdaptUnmarshalParamsInvalidExtraEIP(t *testing.T) {
	cdc := makeCodec()

	legacy := legacytypes.Params{
		EvmDenom:                "aXRP",
		ExtraEIPs:               []string{"ethereum_jpc"},
		ActiveStaticPrecompiles: []string{"0x0000000000000000000000000000000000000800"},
	}

	bz, err := cdc.Marshal(&legacy)
	require.NoError(t, err)

	_, err = vmkeeper.AdaptUnmarshalParams(cdc, bz)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid legacy ExtraEIP")
}

func TestAdaptUnmarshalParamsInvalidBytes(t *testing.T) {
	cdc := makeCodec()

	_, err := vmkeeper.AdaptUnmarshalParams(cdc, []byte{0xFF, 0xFE, 0xFD})
	require.Error(t, err)
	require.Contains(t, err.Error(), "neither current nor legacy schema matched")
}
