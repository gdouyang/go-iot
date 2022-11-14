package eventbus_test

import (
	"fmt"
	"go-iot/codec/eventbus"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatcher(t *testing.T) {
	assert.True(t, true)
	var m1 map[string]string = map[string]string{}
	m1["a"] = "aaa"
	fmt.Println("m1:", m1)

	t1(m1)
	fmt.Println("m1:", m1)

	var m21 map[string]string = map[string]string{}
	m21 = m1
	m21["a"] = "ccc"
	fmt.Println("m21:", m21)

	fmt.Println("m1:", m1)

	var m3 map[string]string
	fmt.Println("m3:", m3)

	match := eventbus.NewAntPathMatcher()
	assert.True(t, match.Match("/a/b/c", "/a/b/c"))
	assert.True(t, match.Match("/a/b/{c}", "/a/b/c"))
	assert.True(t, match.Match("/a/b/{c}/{d}", "/a/b/c/d"))
	assert.True(t, match.Match("/a/b/*/*", "/a/b/c/d"))
}

func t1(m2 map[string]string) {
	m2["a"] = "bbb"
	fmt.Println("m2:", m2)
}
