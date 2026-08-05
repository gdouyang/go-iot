package cluster

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRegistry_ConfigureFindAndKeepalive(t *testing.T) {
	r := newRegistry()
	r.configure(
		ClusterNode{Name: "n1", Url: "http://127.0.0.1:8080", Index: 0},
		[]string{"http://127.0.0.1:8081", "http://127.0.0.1:8080"}, // self url filtered
		"tok",
		true,
	)
	require.True(t, r.Enabled())
	require.Equal(t, "n1", r.LocalID())
	require.Equal(t, 2, r.Size()) // local + 1 peer
	require.Len(t, r.ListPeers(), 1)

	// peer name empty until keepalive
	require.Nil(t, r.FindByName("n2"))

	r.UpsertFromKeepalive(ClusterNode{Name: "n2", Url: "http://127.0.0.1:8081", Index: 1})
	n := r.FindByName("n2")
	require.NotNil(t, n)
	require.Equal(t, "n2", n.Name)
	require.True(t, n.Alive)
	require.False(t, n.LastSeen.IsZero())

	r.MarkPeer("http://127.0.0.1:8081", false, nil)
	n2 := r.FindByName("n2")
	require.NotNil(t, n2)
	require.False(t, n2.Alive)

	all := r.ListAll()
	require.Len(t, all, 2)
	_ = time.Now()
}

func TestRegistry_ListAlivePeers(t *testing.T) {
	r := newRegistry()
	r.configure(ClusterNode{Name: "a", Url: "http://a", Index: 0}, []string{"http://b", "http://c"}, "t", true)
	r.MarkPeer("http://b", true, &ClusterNode{Name: "b", Index: 1})
	alive := r.ListAlivePeers()
	require.Len(t, alive, 1)
	require.Equal(t, "b", alive[0].Name)
}
