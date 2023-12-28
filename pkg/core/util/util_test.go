package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCrc16(t *testing.T) {
	assert.Equal(t, "ce03", ToCrc16Str("0100de03e90302000500"))
	assert.Equal(t, "f2f4", ToCrc16Str("01000000EB0301000a"))
}
