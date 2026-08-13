package models_test

import (
	device "go-iot/pkg/models/device"
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
	matched = device.DeviceIdValid("123abc.Dew")
	assert.False(t, matched)
	matched = device.DeviceIdValid(".123abcDew")
	assert.False(t, matched)
	matched = device.DeviceIdValid("123abcDew.")
	assert.False(t, matched)
	matched = device.DeviceIdValid(".123abcDew.")
	assert.False(t, matched)
}

func TestNormalizeId(t *testing.T) {
	assert.Equal(t, "MYPRODUCT", device.NormalizeId("MyProduct"))
	assert.Equal(t, "DEV-01", device.NormalizeId("  dev-01  "))
	assert.Equal(t, "ABC_123", device.NormalizeId("abc_123"))
}
