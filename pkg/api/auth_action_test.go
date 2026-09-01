package api

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateActionIdIsAdd(t *testing.T) {
	require.Equal(t, "add", CreateAction.Id)
	require.Equal(t, "新增", CreateAction.Name)
}
