package statesync

import (
	"encoding/hex"
	"testing"

	"github.com/nspcc-dev/neo-go/internal/random"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"github.com/stretchr/testify/require"
)

func TestPool_AddRemoveUpdate(t *testing.T) {
	mp := NewPool()

	i1 := []byte{1, 2, 3}
	i1h := util.Uint256{1, 2, 3}
	i2 := []byte{2, 3, 4}
	i2h := util.Uint256{2, 3, 4}
	i3 := []byte{4, 5, 6}
	i3h := util.Uint256{3, 4, 5}
	i4 := []byte{3, 4, 5} // has the same hash as i3
	i5 := []byte{6, 7, 8} // has the same hash as i3
	mapAll := map[util.Uint256][][]byte{i1h: {i1}, i2h: {i2}, i3h: {i4, i3}}

	// No items
	_, ok := mp.TryGet(i1h)
	require.False(t, ok)
	require.False(t, mp.ContainsKey(i1h))
	require.Equal(t, 0, mp.Count())
	require.Equal(t, map[util.Uint256][][]byte{}, mp.GetAll())

	// Add i1, i2, check OK
	mp.Add(i1h, i1)
	mp.Add(i2h, i2)
	itm, ok := mp.TryGet(i1h)
	require.True(t, ok)
	require.Equal(t, [][]byte{i1}, itm)
	require.True(t, mp.ContainsKey(i1h))
	require.True(t, mp.ContainsKey(i2h))
	require.Equal(t, map[util.Uint256][][]byte{i1h: {i1}, i2h: {i2}}, mp.GetAll())
	require.Equal(t, 2, mp.Count())

	// Remove i1 and unexisting item
	mp.Remove(i3h)
	mp.Remove(i1h)
	require.False(t, mp.ContainsKey(i1h))
	require.True(t, mp.ContainsKey(i2h))
	require.Equal(t, map[util.Uint256][][]byte{i2h: {i2}}, mp.GetAll())
	require.Equal(t, 1, mp.Count())

	// Update: remove nothing, add all
	mp.Update(nil, mapAll)
	require.Equal(t, mapAll, mp.GetAll())
	require.Equal(t, 3, mp.Count())
	// Update: remove all, add all
	mp.Update(mapAll, mapAll)
	require.Equal(t, mapAll, mp.GetAll()) // deletion first, addition after that
	require.Equal(t, 3, mp.Count())
	// Update: remove all, add nothing
	mp.Update(mapAll, nil)
	require.Equal(t, map[util.Uint256][][]byte{}, mp.GetAll())
	require.Equal(t, 0, mp.Count())
	// Update: remove several, add several
	mp.Update(map[util.Uint256][][]byte{i1h: {i1}, i2h: {i2}}, map[util.Uint256][][]byte{i2h: {i2}, i3h: {i3}})
	require.Equal(t, map[util.Uint256][][]byte{i2h: {i2}, i3h: {i3}}, mp.GetAll())
	require.Equal(t, 2, mp.Count())

	// Update: remove nothing, add several with same hashes
	mp.Update(nil, map[util.Uint256][][]byte{i3h: {i5, i4}}) // should be sorted by the pool
	require.Equal(t, map[util.Uint256][][]byte{i2h: {i2}, i3h: {i4, i3, i5}}, mp.GetAll())
	require.Equal(t, 2, mp.Count())
	// Update: remove several with same hashes, add nothing
	mp.Update(map[util.Uint256][][]byte{i3h: {i5, i4}}, nil)
	require.Equal(t, map[util.Uint256][][]byte{i2h: {i2}, i3h: {i3}}, mp.GetAll())
	require.Equal(t, 2, mp.Count())
	// Update: remove several with same hashes, add several with same hashes
	mp.Update(map[util.Uint256][][]byte{i3h: {i5, i3}}, map[util.Uint256][][]byte{i3h: {i5, i4}})
	require.Equal(t, map[util.Uint256][][]byte{i2h: {i2}, i3h: {i4, i5}}, mp.GetAll())
	require.Equal(t, 2, mp.Count())
}

func TestPool_GetBatch(t *testing.T) {
	check := func(t *testing.T, limit int, itemsCount int) {
		mp := NewPool()
		for i := 0; i < itemsCount; i++ {
			mp.Add(random.Uint256(), []byte{0x01})
		}
		batch := mp.GetBatch(limit)
		if limit < itemsCount {
			require.Equal(t, limit, len(batch))
		} else {
			require.Equal(t, itemsCount, len(batch))
		}
	}

	t.Run("limit less than items count", func(t *testing.T) {
		check(t, 5, 6)
	})
	t.Run("limit more than items count", func(t *testing.T) {
		check(t, 6, 5)
	})
	t.Run("items count limit", func(t *testing.T) {
		check(t, 5, 5)
	})
}

func TestPool_UpdateUsingSliceFromPool(t *testing.T) {
	mp := NewPool()
	p1, _ := hex.DecodeString("0f0a0f0f0f0f0f0f0104020b02080c0a06050e070b050404060206060d07080602030b04040b050e040406030f0708060c05")
	p2, _ := hex.DecodeString("0f0a0f0f0f0f0f0f01040a0b000f04000b03090b02090b0e040f0d0b060d070e0b0b090b0906080602060c0d0f0e0d04070e")
	p3, _ := hex.DecodeString("0f0a0f0f0f0f0f0f01040b010d01080f050f000a0d0e08060c040b050800050904060f050807080a080c07040d0107080007")
	h, _ := util.Uint256DecodeStringBE("57e197679ef031bf2f0b466b20afe3f67ac04dcff80a1dc4d12dd98dd21a2511")
	mp.Add(h, p1)
	mp.Add(h, p2)
	mp.Add(h, p3)

	toBeRemoved, ok := mp.TryGet(h)
	require.True(t, ok)

	mp.Update(map[util.Uint256][][]byte{h: toBeRemoved}, nil)
	// test that all items were successfully removed.
	require.Equal(t, 0, len(mp.GetAll()))
}
