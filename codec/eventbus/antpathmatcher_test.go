package eventbus_test

import (
	"fmt"
	"go-iot/codec/eventbus"
	"regexp"
	"testing"
	"time"

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
	assert.True(t, match.Match("/abc/123/{name}", "/abc/123/test"))
	assert.True(t, match.Match("/abc/123/{name}/**", "/abc/123/test/1"))
	assert.True(t, match.Match("/abc/123/{name}/{type}", "/abc/123/test/1"))
	assert.True(t, match.Match("/abc/123/{name}/**", "/abc/123/test/1/"))
	assert.True(t, match.Match("/abc/123/{name}/**", "/abc/123/test/1/1"))
	assert.True(t, match.Match("/**", "/abc/123/test/1/1"))
	{
		variables, err := match.ExtractUriTemplateVariables("/abc/123/{name}/{type}", "/abc/123/test/1")
		assert.Nil(t, err)
		assert.Equal(t, "test", variables["name"])
		assert.Equal(t, "1", variables["type"])
	}
	assert.True(t, regexp.MustCompile("(?i)abc").Match([]byte("Abc")))
	assert.True(t, regexp.MustCompile("(?i)abc").Match([]byte("abc")))

	{
		_, err := match.ExtractUriTemplateVariables("/abc/123/{name}/{type}", "/abc/123/test/1/")
		assert.NotNil(t, err)
	}
	assert.False(t, match.Match("/abc/123/{name}/{type}", "/abc/123/test/1/"))
	assert.False(t, match.Match("/abc/123/*/*", "/abc/123/test/1/"))
	assert.False(t, match.Match("/", "/abc/123/test/1"))
	assert.False(t, regexp.MustCompile("abc").Match([]byte("Abc")))
}

func t1(m2 map[string]string) {
	m2["a"] = "bbb"
	fmt.Println("m2:", m2)
}

func TestThreadSafe(t *testing.T) {
	match := eventbus.NewAntPathMatcher()
	go _thread(t, match)
	go _thread(t, match)
	go _thread(t, match)
	time.Sleep(time.Second * 2)
}

func _thread(t *testing.T, match *eventbus.AntPathMatcher) {
	for i := 0; i < 100; i++ {
		typ := fmt.Sprintf("%v", i)
		variables, err := match.ExtractUriTemplateVariables("/abc/123/{name"+typ+"}/{type}", "/abc/123/test/"+typ)
		assert.Nil(t, err)
		assert.Equal(t, "test", variables["name"+typ])
		assert.Equal(t, typ, variables["type"])
	}
	fmt.Println("thread ")
}
