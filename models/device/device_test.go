package models_test

import (
	device "go-iot/models/device"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestId(t *testing.T) {
	matched := device.DeviceIdValid("123")
	assert.True(t, matched)
	matched = device.DeviceIdValid("123abcDew-_")
	assert.True(t, matched)
	matched = device.DeviceIdValid("1222-112")
	assert.True(t, matched)
	matched = device.DeviceIdValid("12abee22_112")
	assert.True(t, matched)
	matched = device.DeviceIdValid("ABCD-123-abc_")
	assert.True(t, matched)
	matched = device.DeviceIdValid("123abcDew-_@")
	assert.False(t, matched)
}
