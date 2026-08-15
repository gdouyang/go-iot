package orm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type unregisteredSample struct {
	Id string
}

func TestIsRegisteredFalseForUnknownType(t *testing.T) {
	require.False(t, IsRegistered(unregisteredSample{}))
	require.False(t, IsRegistered(&unregisteredSample{}))
}
