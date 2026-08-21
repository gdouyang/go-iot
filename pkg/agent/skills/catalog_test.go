package skills

import (
	"strings"
	"testing"

	"go-iot/pkg/agent/codecdoc"

	"github.com/stretchr/testify/require"
)

func TestListMatchesCodecdocTopics(t *testing.T) {
	got := map[string]struct{}{}
	for _, s := range List() {
		got[s.Topic] = struct{}{}
		require.NotEmpty(t, s.Name)
		require.NotEmpty(t, s.Description)
	}
	for _, topic := range codecdoc.Topics() {
		_, ok := got[topic]
		require.True(t, ok, topic)
	}
}

func TestResolveAliases(t *testing.T) {
	for _, name := range []string{"codec-mqtt-broker", "MQTT_BROKER", "mqtt-broker"} {
		s, ok := Resolve(name)
		require.True(t, ok, name)
		require.Equal(t, "codec-mqtt-broker", s.Name)
		require.Equal(t, "MQTT_BROKER", s.Topic)
	}
	s, ok := Resolve("splitFunc")
	require.True(t, ok)
	require.Equal(t, "split", s.Name)
	_, ok = Resolve("MQTT")
	require.False(t, ok)
}

func TestGetReturnsCodecdocBody(t *testing.T) {
	s, md, err := Get("HTTP_SERVER")
	require.NoError(t, err)
	require.Equal(t, "codec-http-server", s.Name)
	want, err := codecdoc.Get("HTTP_SERVER")
	require.NoError(t, err)
	require.Equal(t, want, md)

	_, _, err = Get("MQTT")
	require.Error(t, err)
}

func TestCatalogMarkdownListsEverySkill(t *testing.T) {
	md := CatalogMarkdown()
	require.True(t, strings.HasPrefix(md, "<available_skills>\n"))
	require.True(t, strings.HasSuffix(md, "</available_skills>\n"))
	for _, s := range List() {
		require.Contains(t, md, "<name>"+xmlEscape(s.Name)+"</name>")
		require.Contains(t, md, "<description>"+xmlEscape(s.Description)+"</description>")
	}
}

func TestXmlEscape(t *testing.T) {
	require.Equal(t, "a&amp;b &lt;c&gt;", xmlEscape("a&b <c>"))
	require.Equal(t, "&amp;lt;", xmlEscape("&lt;"))
}

func TestMqttBrokerSkillsDistinguishTopology(t *testing.T) {
	own, ok := Resolve("MQTT_BROKER")
	require.True(t, ok)
	share, ok := Resolve("GOIOT_MQTT_BROKER")
	require.True(t, ok)
	require.Contains(t, own.Description, "独占端口")
	require.Contains(t, share.Description, "1883")
	require.Contains(t, share.Description, "clientId=设备ID")
	require.NotContains(t, own.Description, "1883")
}
