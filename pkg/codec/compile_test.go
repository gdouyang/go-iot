package codec_test

import (
	"testing"

	"go-iot/pkg/codec"
	"go-iot/pkg/core"

	"github.com/stretchr/testify/require"
)

func TestCompileOnlyDoesNotReplaceLiveCodec(t *testing.T) {
	productID := "COMPILE-ONLY-LIVE"
	t.Cleanup(func() {
		core.RegCodec(productID, nil)
	})

	live, err := codec.NewScriptCodec(productID, `function OnMessage(ctx) { return 1 }`)
	require.NoError(t, err)
	require.Same(t, live, core.GetCodec(productID))

	require.Error(t, codec.CompileOnly("function broken("))
	require.Same(t, live, core.GetCodec(productID))

	require.NoError(t, codec.CompileOnly(`function OnMessage(ctx) { return 2 }`))
	require.Same(t, live, core.GetCodec(productID))
}

func TestCompileOnlyRejectsEmptyScript(t *testing.T) {
	require.Error(t, codec.CompileOnly(""))
}
