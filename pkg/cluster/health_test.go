package cluster

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHealth_DisabledGreen(t *testing.T) {
	r := newRegistry()
	r.configure(ClusterNode{Name: "solo", Url: "http://s", Index: 0}, nil, "", false)
	old := defaultRegistry
	defaultRegistry = r
	t.Cleanup(func() { defaultRegistry = old })

	h := Health()
	require.Equal(t, HealthGreen, h.Status)
	require.False(t, h.Enabled)
	require.Equal(t, 1, h.NumberOfNodes)
	require.Equal(t, 1, h.NumberOfNodesAlive)
	require.Equal(t, 0, h.NumberOfPeers)
}

func TestHealth_AllPeersAliveGreen(t *testing.T) {
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

	h := Health()
	require.Equal(t, HealthGreen, h.Status)
	require.True(t, h.Enabled)
	require.Equal(t, 2, h.NumberOfNodes)
	require.Equal(t, 2, h.NumberOfNodesAlive)
	require.Equal(t, 1, h.NumberOfPeersAlive)
	require.Equal(t, 0, h.UnnamedPeers)
	require.Len(t, h.Nodes, 2)
}

func TestHealth_PeerDownYellow(t *testing.T) {
	r := newRegistry()
	r.configure(
		ClusterNode{Name: "n0", Url: "http://n0", Index: 0},
		[]string{"http://n1", "http://n2"},
		"tok",
		true,
	)
	r.UpsertFromKeepalive(ClusterNode{Name: "n1", Url: "http://n1", Index: 1})
	// n2 configured but never alive
	old := defaultRegistry
	defaultRegistry = r
	t.Cleanup(func() { defaultRegistry = old })

	h := Health()
	require.Equal(t, HealthYellow, h.Status)
	require.Equal(t, 1, h.NumberOfPeersAlive)
	require.Equal(t, 2, h.NumberOfPeers)
}

func TestHealth_AllPeersDownRed(t *testing.T) {
	r := newRegistry()
	r.configure(
		ClusterNode{Name: "n0", Url: "http://n0", Index: 0},
		[]string{"http://n1"},
		"tok",
		true,
	)
	old := defaultRegistry
	defaultRegistry = r
	t.Cleanup(func() { defaultRegistry = old })

	h := Health()
	require.Equal(t, HealthRed, h.Status)
	require.Equal(t, 0, h.NumberOfPeersAlive)
}

func TestHealth_EmptyLocalNameRed(t *testing.T) {
	r := newRegistry()
	r.configure(
		ClusterNode{Name: "", Url: "http://n0", Index: 0},
		[]string{"http://n1"},
		"tok",
		true,
	)
	old := defaultRegistry
	defaultRegistry = r
	t.Cleanup(func() { defaultRegistry = old })

	h := Health()
	require.Equal(t, HealthRed, h.Status)
}
