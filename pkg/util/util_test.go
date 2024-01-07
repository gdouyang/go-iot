package util

import (
	"go-iot/pkg/logger"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCrc16(t *testing.T) {
	logger.InitNop()
	v, err := ToCrc16Str("0100de03e90302000500")
	assert.Nil(t, err)
	assert.Equal(t, "ce03", v)
	v, err = ToCrc16Str("01000000EB0301000a")
	assert.Nil(t, err)
	assert.Equal(t, "f2f4", err)
}
