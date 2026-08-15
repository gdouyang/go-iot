package codec

// CompileOnly compiles a JS codec script into a temporary VmPool and closes it.
// It must not RegCodec / RegDeviceLifeCycle or close any live product pool.
func CompileOnly(script string) error {
	pool, err := NewVmPool1(script, 1, "")
	if err != nil {
		return err
	}
	pool.Close()
	return nil
}
