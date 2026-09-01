package skills

import (
	"fmt"
	"strings"

	"go-iot/pkg/agent/codecdoc"
)

// Skill is one codec knowledge pack: cheap catalog entry + on-demand markdown body.
type Skill struct {
	Name        string
	Description string
	Topic       string
}

var catalog = []Skill{
	{Name: "codec-mqtt-broker", Description: "MQTT_BROKER：产品自己起一套 Broker、独占端口；连到该端口的设备都归这个产品。用户说自建 broker、自定义 MQTT Broker 时用。和平台内置 MQTT 不是同一个。", Topic: "MQTT_BROKER"},
	{Name: "codec-goiot-mqtt-broker", Description: "GOIOT_MQTT_BROKER：全平台共用一个 Broker（默认 1883），产品不另占端口；设备用 clientId=设备ID 接入，平台再反查所属产品。用户说平台 MQTT、内置 broker 时用。", Topic: "GOIOT_MQTT_BROKER"},
	{Name: "codec-tcp-server", Description: "TCP_SERVER：产品单独起 TCP 监听，设备作为客户端连上来，适合自定义二进制/文本长连接。粘包拆包另 load split。", Topic: "TCP_SERVER"},
	{Name: "codec-http-server", Description: "HTTP_SERVER：产品单独起 HTTP，设备用 HTTP 上报；协议无状态，在线状态靠脚本查询，不是长连接保活。", Topic: "HTTP_SERVER"},
	{Name: "codec-websocket-server", Description: "WEBSOCKET_SERVER：产品单独起 WebSocket，设备连上来做双向实时通信。", Topic: "WEBSOCKET_SERVER"},
	{Name: "codec-coap-server", Description: "COAP_SERVER：产品单独起 CoAP，适合受限设备用 CoAP 上报；一次请求一次响应。", Topic: "COAP_SERVER"},
	{Name: "codec-mqtt-client", Description: "MQTT_CLIENT：平台作为客户端去连外部 Broker，设备挂在那个 Broker 上。不是自建/内置 Broker（那是 MQTT_BROKER / GOIOT_MQTT_BROKER）。", Topic: "MQTT_CLIENT"},
	{Name: "codec-tcp-client", Description: "TCP_CLIENT：平台作为 TCP 客户端去连设备或设备侧服务，不是设备连上来。", Topic: "TCP_CLIENT"},
	{Name: "codec-modbus", Description: "MODBUS：平台作为 Modbus 主站。采集只用产品点表+采集组（组包读寄存器）。可选 OnMessage 二次处理 GetMessage()={source,groupId,properties}，无则采集器直存。写命令走 OnInvoke。没有 OnConnect。", Topic: "MODBUS"},
	{Name: "split", Description: "TCP 粘包拆包。写 TCP_SERVER/TCP_CLIENT 且报文要按分隔符或定长切开时用 splitFunc。", Topic: "SPLIT"},
	{Name: "cron", Description: "定时任务 cron 表达式（分 时 日 月 周）。产品或规则要按时间触发时用。", Topic: "CRON"},
}

var extraAliases = map[string]string{
	"SPLITFUNC": "split",
}

var byKey map[string]Skill

func init() {
	byKey = make(map[string]Skill, len(catalog)*4)
	for _, s := range catalog {
		for _, k := range []string{s.Name, s.Topic} {
			byKey[norm(k)] = s
		}
	}
	for alias, name := range extraAliases {
		s, ok := byKey[norm(name)]
		if !ok {
			continue
		}
		byKey[norm(alias)] = s
	}
}

func norm(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	return strings.ReplaceAll(s, "-", "_")
}

func List() []Skill {
	out := make([]Skill, len(catalog))
	copy(out, catalog)
	return out
}

func Resolve(name string) (Skill, bool) {
	s, ok := byKey[norm(name)]
	return s, ok
}

func Get(name string) (Skill, string, error) {
	s, ok := Resolve(name)
	if !ok {
		return Skill{}, "", fmt.Errorf("unknown skill %s", name)
	}
	md, err := codecdoc.Get(s.Topic)
	if err != nil {
		return s, "", err
	}
	return s, md, nil
}

func CatalogMarkdown() string {
	var b strings.Builder
	b.WriteString("<available_skills>\n")
	for _, s := range catalog {
		b.WriteString("<skill>\n<name>")
		b.WriteString(xmlEscape(s.Name))
		b.WriteString("</name>\n<description>")
		b.WriteString(xmlEscape(s.Description))
		b.WriteString("</description>\n</skill>\n")
	}
	b.WriteString("</available_skills>\n")
	return b.String()
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func Summaries() []map[string]string {
	out := make([]map[string]string, 0, len(catalog))
	for _, s := range catalog {
		out = append(out, map[string]string{"name": s.Name, "description": s.Description})
	}
	return out
}
