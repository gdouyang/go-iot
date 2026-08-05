package cluster

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOwnerIndex_Stable(t *testing.T) {
	r := newRegistry()
	r.configure(
		ClusterNode{Name: "n0", Url: "http://n0", Index: 0},
		[]string{"http://n1", "http://n2"},
		"tok",
		true,
	)
	r.UpsertFromKeepalive(ClusterNode{Name: "n1", Url: "http://n1", Index: 1})
	r.UpsertFromKeepalive(ClusterNode{Name: "n2", Url: "http://n2", Index: 2})

	old := defaultRegistry
	defaultRegistry = r
	t.Cleanup(func() { defaultRegistry = old })

	require.Equal(t, 3, Size())
	idx := OwnerIndex("device-abc")
	require.GreaterOrEqual(t, idx, 0)
	require.Less(t, idx, 3)
	require.Equal(t, idx, OwnerIndex("device-abc"))
}

func TestResolveShardOwner_LocalAndRemote(t *testing.T) {
	r := newRegistry()
	r.configure(
		ClusterNode{Name: "n0", Url: "http://n0", Index: 0},
		[]string{"http://n1"},
		"tok",
		true,
	)
	r.UpsertFromKeepalive(ClusterNode{Name: "n1", Url: "http://n1", Index: 1})
	old := defaultRegistry
	defaultRegistry = r
	t.Cleanup(func() { defaultRegistry = old })

	var keyLocal, keyRemote string
	for i := 0; i < 5000; i++ {
		k := fmt.Sprintf("device-%d", i)
		switch OwnerIndex(k) {
		case 0:
			if keyLocal == "" {
				keyLocal = k
			}
		case 1:
			if keyRemote == "" {
				keyRemote = k
			}
		}
		if keyLocal != "" && keyRemote != "" {
			break
		}
	}
	require.NotEmpty(t, keyLocal)
	require.NotEmpty(t, keyRemote)

	local, remote, err := ResolveShardOwner(keyLocal)
	require.NoError(t, err)
	require.True(t, local)
	require.Nil(t, remote)
	require.True(t, Shard(keyLocal))

	local, remote, err = ResolveShardOwner(keyRemote)
	require.NoError(t, err)
	require.False(t, local)
	require.NotNil(t, remote)
	require.Equal(t, "n1", remote.Name)
	require.False(t, Shard(keyRemote))
}

func TestResolveShardOwner_Disabled(t *testing.T) {
	r := newRegistry()
	r.configure(ClusterNode{Name: "solo", Url: "http://s", Index: 0}, nil, "", false)
	old := defaultRegistry
	defaultRegistry = r
	t.Cleanup(func() { defaultRegistry = old })

	local, remote, err := ResolveShardOwner("any")
	require.NoError(t, err)
	require.True(t, local)
	require.Nil(t, remote)
}

func TestResolveShardOwner_UnknownPeer(t *testing.T) {
	r := newRegistry()
	// peer configured but never keepalive → no Name/Index match for index 1
	r.configure(
		ClusterNode{Name: "n0", Url: "http://n0", Index: 0},
		[]string{"http://n1"},
		"tok",
		true,
	)
	old := defaultRegistry
	defaultRegistry = r
	t.Cleanup(func() { defaultRegistry = old })

	// peer Index still 0 default — OwnerIndex 1 won't find node with Index 1
	var keyRemote string
	for i := 0; i < 5000; i++ {
		k := fmt.Sprintf("x-%d", i)
		if OwnerIndex(k) == 1 {
			keyRemote = k
			break
		}
	}
	require.NotEmpty(t, keyRemote)
	_, _, err := ResolveShardOwner(keyRemote)
	require.Error(t, err)
}
