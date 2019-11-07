package util

import (
	"os"
	"path/filepath"
)

func FileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func FileSize(filename string) int64 {
	var size int64
	filepath.Walk(filename, func(path string, f os.FileInfo, err error) error {
		size = f.Size()
		return nil
	})
	return size
}
