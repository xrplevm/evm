package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	legacytypes "github.com/cosmos/evm/rpc/types/legacy"
	"github.com/cosmos/evm/x/vm/types"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/testutil"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
)

// newParamsKeeper builds a minimal Keeper with only the fields needed by
// GetParams / SetParams (cdc + storeKey), avoiding NewKeeper which sets
// the global chainConfig singleton and panics on repeated calls.
func newParamsKeeper(t *testing.T) (Keeper, testutil.TestContext) {
	t.Helper()
	key := storetypes.NewKVStoreKey(types.StoreKey)
	tkey := storetypes.NewTransientStoreKey("transient_test")
	testCtx := testutil.DefaultContextWithDB(t, key, tkey)
	encCfg := moduletestutil.MakeTestEncodingConfig()
	k := Keeper{
		cdc:      encCfg.Codec,
		storeKey: key,
	}
	return k, testCtx
}

func TestGetParamsEmptyStore(t *testing.T) {
	k, testCtx := newParamsKeeper(t)
	params := k.GetParams(testCtx.Ctx)
	require.Equal(t, types.Params{}, params)
}

func TestGetParamsCurrentSchema(t *testing.T) {
	k, testCtx := newParamsKeeper(t)
	ctx := testCtx.Ctx

	original := types.Params{
		EvmDenom:                "aXRP",
		ExtraEIPs:               []int64{3855},
		EVMChannels:             []string{"channel-0"},
		ActiveStaticPrecompiles: []string{"0x0000000000000000000000000000000000000800"},
		AccessControl: types.AccessControl{
			Create: types.AccessControlType{
				AccessType: types.AccessTypePermissionless,
			},
			Call: types.AccessControlType{
				AccessType: types.AccessTypePermissionless,
			},
		},
	}

	// Write current-schema bytes directly to the store.
	bz, err := k.cdc.Marshal(&original)
	require.NoError(t, err)
	ctx.KVStore(k.storeKey).Set(types.KeyPrefixParams, bz)

	params := k.GetParams(ctx)
	require.Equal(t, original.EvmDenom, params.EvmDenom)
	require.Equal(t, original.ExtraEIPs, params.ExtraEIPs)
	require.Equal(t, original.EVMChannels, params.EVMChannels)
	require.Equal(t, original.ActiveStaticPrecompiles, params.ActiveStaticPrecompiles)
}

func TestGetParamsLegacySchema(t *testing.T) {
	k, testCtx := newParamsKeeper(t)
	ctx := testCtx.Ctx

	legacy := legacytypes.Params{
		EvmDenom:                "aXRP",
		ExtraEIPs:               []string{"ethereum_3855"},
		EVMChannels:             []string{"channel-0"},
		ActiveStaticPrecompiles: []string{"0x0000000000000000000000000000000000000800"},
		AccessControl: legacytypes.AccessControl{
			Create: legacytypes.AccessControlType{
				AccessType: legacytypes.AccessTypePermissionless,
			},
			Call: legacytypes.AccessControlType{
				AccessType: legacytypes.AccessTypePermissionless,
			},
		},
	}

	// Write legacy-encoded bytes directly to the store, simulating
	// what IAVL returns when querying a pre-v9 height.
	bz, err := k.cdc.Marshal(&legacy)
	require.NoError(t, err)
	ctx.KVStore(k.storeKey).Set(types.KeyPrefixParams, bz)

	params := k.GetParams(ctx)
	require.Equal(t, "aXRP", params.EvmDenom)
	require.Equal(t, []int64{3855}, params.ExtraEIPs)
	require.Equal(t, []string{"channel-0"}, params.EVMChannels)
	require.Equal(t, []string{"0x0000000000000000000000000000000000000800"}, params.ActiveStaticPrecompiles)
	require.Equal(t, uint64(0), params.HistoryServeWindow)
	require.Nil(t, params.ExtendedDenomOptions)
}

func TestGetParamsGarbagePanics(t *testing.T) {
	k, testCtx := newParamsKeeper(t)
	ctx := testCtx.Ctx

	// Write garbage bytes — GetParams should panic since AdaptUnmarshalParams
	// returns an error and GetParams wraps it in panic().
	ctx.KVStore(k.storeKey).Set(types.KeyPrefixParams, []byte{0xFF, 0xFE, 0xFD})

	require.Panics(t, func() {
		k.GetParams(ctx)
	})
}
