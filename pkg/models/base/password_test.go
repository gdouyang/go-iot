package base

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHashAndCheckBcrypt(t *testing.T) {
	hash, err := HashPassword("secret123")
	require.NoError(t, err)
	require.True(t, IsBcryptHash(hash))

	ok, upgrade := CheckPassword(hash, "admin", "secret123")
	require.True(t, ok)
	require.False(t, upgrade)

	ok, _ = CheckPassword(hash, "admin", "wrong")
	require.False(t, ok)
}

func TestLegacyMD5AndLazyUpgradeFlag(t *testing.T) {
	username := "admin"
	plain := "123456"
	legacy := LegacyMD5Hash(username, plain)
	require.False(t, IsBcryptHash(legacy))
	require.Len(t, legacy, 32)

	ok, upgrade := CheckPassword(legacy, username, plain)
	require.True(t, ok)
	require.True(t, upgrade)

	ok, upgrade = CheckPassword(legacy, username, "nope")
	require.False(t, ok)
	require.False(t, upgrade)
}

func TestHashPasswordEmpty(t *testing.T) {
	_, err := HashPassword("")
	require.Error(t, err)
}

func TestBcryptPrefixVariants(t *testing.T) {
	// GenerateFromPassword 通常产出 $2a$
	h, err := HashPassword("x")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(h, "$2"))
}
