package eventbus_test

import (
	"fmt"
	"go-iot/pkg/eventbus"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMatcher(t *testing.T) {
	assert.True(t, true)

	match := eventbus.NewAntPathMatcher()
	assert.True(t, match.Match("/a/b/c", "/a/b/c"))
	assert.True(t, match.Match("/a/b/{c}", "/a/b/c"))
	assert.True(t, match.Match("/a/b/{c}/{d}", "/a/b/c/d"))
	assert.True(t, match.Match("/a/b/*/*", "/a/b/c/d11"))
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
