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

func TestScriptVMPoolSizeDefaultAndClamp(t *testing.T) {
	prev := Global
	t.Cleanup(func() { Global = prev })

	Global = nil
	require.Equal(t, DefaultScriptVMPoolSize, ScriptVMPoolSize())

	Global = &Options{Script: Script{VMPoolSize: 0}}
	require.Equal(t, DefaultScriptVMPoolSize, ScriptVMPoolSize())

	Global = &Options{Script: Script{VMPoolSize: 200}}
	require.Equal(t, 200, ScriptVMPoolSize())

	Global = &Options{Script: Script{VMPoolSize: 9999}}
	require.Equal(t, MaxScriptVMPoolSize, ScriptVMPoolSize())
}

func TestParseScriptVMPoolSizeFromYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
script:
  vm-pool-size: 80
`), 0o644))
	prev := Global
	t.Cleanup(func() { Global = prev })
	opt := New()
	opt.ConfigFile = path
	old := os.Args
	os.Args = []string{"go-iot"}
	t.Cleanup(func() { os.Args = old })
	_, err := opt.Parse()
	require.NoError(t, err)
	require.Equal(t, 80, opt.Script.VMPoolSize)
	require.Equal(t, 80, ScriptVMPoolSize())
}

func TestReadLogLevelFromFileMissing(t *testing.T) {
	opt := &Options{ConfigFile: filepath.Join(t.TempDir(), "no-such.yaml")}
	_, err := opt.ReadLogLevelFromFile()
	require.Error(t, err)
}
