package option

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseHasNoAgentConfigBlock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte("api-addr: :8088\n"), 0o644))
	opt := New()
	opt.ConfigFile = path
	old := os.Args
	os.Args = []string{"go-iot"}
	t.Cleanup(func() { os.Args = old })
	_, err := opt.Parse()
	require.NoError(t, err)
	raw := opt.YAML()
	require.NotContains(t, raw, "\nagent:")
	require.NotContains(t, raw, "agent.enabled")
	require.NotContains(t, raw, "MASTER_KEY")
	require.NotContains(t, raw, "XAI_API_KEY")
}
