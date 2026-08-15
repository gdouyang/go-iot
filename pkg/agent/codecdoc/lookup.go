package codecdoc

import (
	"embed"
	"fmt"
	"sort"
	"strings"
)

//go:embed *.md
var fs embed.FS

var aliases = map[string]string{
	"MQTT_BROKER":       "mqtt_broker.md",
	"GOIOT_MQTT_BROKER": "goiot_mqtt_broker.md",
	"MQTT_CLIENT":       "mqtt_client.md",
	"TCP_SERVER":        "tcp_server.md",
	"TCP_CLIENT":        "tcp_client.md",
	"HTTP_SERVER":       "http_server.md",
	"WEBSOCKET_SERVER":  "websocket_server.md",
	"MODBUS":            "modbus.md",
	"COAP_SERVER":       "coap_server.md",
	"SPLIT":             "split.md",
	"SPLITFUNC":         "split.md",
	"CRON":              "cron.md",
}

func Topics() []string {
	out := make([]string, 0, len(aliases))
	seen := map[string]struct{}{}
	for k, file := range aliases {
		if _, ok := seen[file]; ok {
			continue
		}
		if k == "SPLITFUNC" {
			continue
		}
		seen[file] = struct{}{}
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func Get(topic string) (string, error) {
	key := strings.ToUpper(strings.TrimSpace(topic))
	key = strings.ReplaceAll(key, "-", "_")
	file, ok := aliases[key]
	if !ok {
		return "", fmt.Errorf("unknown codec doc topic %s", topic)
	}
	b, err := fs.ReadFile(file)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
