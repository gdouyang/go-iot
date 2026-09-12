package codec_test

import (
	"testing"

	"go-iot/pkg/codec"
	"go-iot/pkg/core"
	"go-iot/pkg/logger"

	"github.com/stretchr/testify/require"
)

func TestDryRunOnMessageCapturesLogAndProperties(t *testing.T) {
	logger.InitNop()
	script := `
function OnMessage(context) {
  console.log("got " + context.MsgToString())
  context.SaveProperties({temp: 21})
  context.GetSession().Send("ack")
}
`
	live, err := core.NewCodec(core.Script_Codec, "dry-live", `function OnMessage(c){ console.log("live") }`)
	require.NoError(t, err)

	res := codec.DryRun(codec.DryRunOpts{
		ProductId: "dry-p1",
		Script:    script,
		Hook:      "OnMessage",
		Payload:   []byte(`{"a":1}`),
		DeviceId:  "dev-1",
	})
	require.True(t, res.Ok, res.Error)
	require.NotEmpty(t, res.Logs)
	require.Contains(t, res.Logs[0]["data"], "got")
	require.Len(t, res.SavedProperties, 1)
	require.EqualValues(t, 21, res.SavedProperties[0]["temp"])
	require.NotEmpty(t, res.SessionWrites)

	require.Same(t, live, core.GetCodec("dry-live"))
}

func TestDryRunDoesNotOnlineOrReplaceCodec(t *testing.T) {
	logger.InitNop()
	res := codec.DryRun(codec.DryRunOpts{
		ProductId: "dry-p2",
		Script: `
function OnMessage(context) {
  context.DeviceOnline("should-not-exist")
}
`,
		Payload: []byte("x"),
	})
	require.True(t, res.Ok, res.Error)
	require.Equal(t, []string{"should-not-exist"}, res.DeviceOnline)
	require.Nil(t, core.GetSession("should-not-exist"))
	require.Nil(t, core.GetCodec("dry-p2"))
}

func TestDryRunMissingHook(t *testing.T) {
	logger.InitNop()
	res := codec.DryRun(codec.DryRunOpts{
		ProductId: "dry-p3",
		Script:    `function OnConnect(c){}`,
		Hook:      "OnMessage",
		Payload:   []byte("{}"),
	})
	require.False(t, res.Ok)
	require.True(t, res.FunctionMissing)
}

func TestDryRunOnInvokeGetMessage(t *testing.T) {
	logger.InitNop()
	res := codec.DryRun(codec.DryRunOpts{
		ProductId:  "dry-p4",
		Script:     `function OnInvoke(context) { console.log(context.GetMessage().FunctionId); context.ReplyOk() }`,
		Hook:       "OnInvoke",
		FunctionId: "setTemp",
		Data:       map[string]any{"v": 1},
		DeviceId:   "d",
	})
	require.True(t, res.Ok, res.Error)
	require.Contains(t, res.Logs[0]["data"], "setTemp")
	require.Equal(t, []string{"ok"}, res.Replies)
}

func TestDryRunCompileError(t *testing.T) {
	logger.InitNop()
	res := codec.DryRun(codec.DryRunOpts{ProductId: "dry-p5", Script: "function OnMessage("})
	require.False(t, res.Ok)
	require.NotEmpty(t, res.Error)
}
