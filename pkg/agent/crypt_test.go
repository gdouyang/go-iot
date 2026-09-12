package agent

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSealRevealRoundTrip(t *testing.T) {
	sealed, err := SealAPIKey("sk-hello-world")
	require.NoError(t, err)
	require.True(t, IsSealedKey(sealed))
	require.NotContains(t, sealed, "sk-hello")
	require.Equal(t, "sk-hello-world", RevealAPIKey(sealed))
	require.Equal(t, "plain", RevealAPIKey("plain"))
	require.Equal(t, "", RevealAPIKey(""))
}

func TestSealIdempotent(t *testing.T) {
	sealed, err := SealAPIKey("k")
	require.NoError(t, err)
	again, err := SealAPIKey(sealed)
	require.NoError(t, err)
	require.Equal(t, sealed, again)
}
