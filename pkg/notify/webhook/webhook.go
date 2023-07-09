package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go-iot/pkg/notify"
	"io/ioutil"
	"net/http"
	"time"

	logs "go-iot/pkg/logger"
)

func init() {
	notify.RegNotify(func() notify.Notify {
		return &WebHookNotify{}
	})
}

// NotifyConfig is the webhook notification configuration
type WebHookNotify struct {
	WebhookURL string `yaml:"webhook" json:"webhook"`
	name       string `json:"-"`
}

func (c *WebHookNotify) Kind() string {
	return "webhook"
}

func (c *WebHookNotify) Name() string {
	return "WebHook"
}

func (c *WebHookNotify) ParseTemplate(data map[string]interface{}) string {
	b, err := json.Marshal(data)
	if err != nil {
		logs.Warnf("webhook ParseTemplate error: %v", err)
	}

	return string(b)
}

func (c *WebHookNotify) FromJson(config notify.NotifyConfig) error {
	err := json.Unmarshal([]byte(config.Config), c)
	if err != nil {
		return err
	}
	c.name = config.Name
	return err
}

func (c *WebHookNotify) Meta() []map[string]string {
	var m []map[string]string = []map[string]string{
		{"name": "webhook", "type": "string", "required": "true", "title": "Webhook", "desc": "The Webhook URL"},
	}
	return m
}

// post to an 'Webhook' url in Remote Server.
func (c *WebHookNotify) Notify(message string) error {
	msgContent := message
	req, err := http.NewRequest(http.MethodPost, c.WebhookURL, bytes.NewBuffer([]byte(msgContent)))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Close = true

	client := &http.Client{Timeout: time.Duration(time.Second * 3)}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		if resp.Body != nil {
			defer resp.Body.Close()
		}
		buf, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("[%s / %s] - Error response from WebHook [%d] - [%s]",
			c.Kind(), c.Name(), resp.StatusCode, string(buf))
	}
	return nil
}
