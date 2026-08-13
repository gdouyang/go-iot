package option

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadLogLevelFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
logs:
  level: warn
  reload-interval: 10
`), 0o644))

	opt := &Options{ConfigFile: path}
	level, err := opt.ReadLogLevelFromFile()
	require.NoError(t, err)
	require.Equal(t, "warn", level)
}

func TestReadLogLevelFromFileDefaultInfo(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
api-addr: :8088
`), 0o644))

	opt := &Options{ConfigFile: path}
	level, err := opt.ReadLogLevelFromFile()
	require.NoError(t, err)
	require.Equal(t, "info", level)
}

func TestReadLogLevelFromFileMissing(t *testing.T) {
	opt := &Options{ConfigFile: filepath.Join(t.TempDir(), "no-such.yaml")}
	_, err := opt.ReadLogLevelFromFile()
	require.Error(t, err)
}
