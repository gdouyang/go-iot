package dingtalk

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"go-iot/pkg/notify"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/beego/beego/v2/core/logs"
)

func init() {
	notify.RegNotify(func() notify.Notify {
		return &DingtalkNotify{}
	})
}

// NotifyConfig is the dingtalk notification configuration
type DingtalkNotify struct {
	WebhookURL  string             `yaml:"webhook" json:"webhook"`
	SignSecret  string             `yaml:"secret,omitempty" json:"secret,omitempty"`
	name        string             `json:"-"`
	title       string             `json:"-"`
	msgTemplate string             `json:"-"`
	template    *template.Template `json:"-"`
}

func (c *DingtalkNotify) Kind() string {
	return "dingtalk"
}

func (c *DingtalkNotify) Name() string {
	return c.name
}

func (c *DingtalkNotify) Title() string {
	return c.title
}

func (c *DingtalkNotify) ParseTemplate(data map[string]interface{}) string {
	var result bytes.Buffer
	if err := c.template.Execute(&result, data); err != nil {
		logs.Error(err)
		return c.msgTemplate
	}
	return result.String()
}

func (c *DingtalkNotify) FromJson(config notify.NotifyConfig) error {
	err := json.Unmarshal([]byte(config.Config), c)
	if err != nil {
		return err
	}
	c.name = config.Name
	tpl := map[string]string{}
	err = json.Unmarshal([]byte(config.Template), &tpl)
	c.title = tpl["title"]
	msgTemplate := ""
	if str, ok := tpl["text"]; ok {
		msgTemplate = str
	}
	c.msgTemplate = msgTemplate
	msgTemplate = strings.ReplaceAll(msgTemplate, "${", "${.")
	tpl1 := template.New("").Delims("${", "}")
	c.template = template.Must(tpl1.Parse(msgTemplate))
	return err
}

func (c *DingtalkNotify) Meta() []map[string]string {
	var m []map[string]string = []map[string]string{
		{"name": "webhook", "type": "string", "required": "true", "title": "Webhook", "desc": "The Dingtalk Robot Webhook URL"},
		{"name": "secret", "type": "string", "required": "false", "title": "Secret", "desc": "The Dingtalk Robot Secret"},
	}
	return m
}

// SendDingtalkNotification will post to an 'Robot Webhook' url in Dingtalk Apps. It accepts
// some text and the Dingtalk robot will send it in group.
func (c *DingtalkNotify) Notify(subject string, message string) error {
	title := "**" + c.title + "**"
	// It will be better to escape the msg.
	msgContent := fmt.Sprintf(`
{
	"msgtype": "markdown",
	"markdown": {
		"title": "%s",
		"text": "%s"
	}
}
`, title, message)
	req, err := http.NewRequest(http.MethodPost, c.addSign(c.WebhookURL, c.SignSecret), bytes.NewBuffer([]byte(msgContent)))
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
	if resp.Body != nil {
		defer resp.Body.Close()
	}

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	ret := make(map[string]interface{})
	err = json.Unmarshal(buf, &ret)
	if err != nil || ret["errmsg"] != "ok" {
		return fmt.Errorf("[%s / %s] - Error response from Dingtalk [%d] - [%s]",
			c.Kind(), c.Name(), resp.StatusCode, string(buf))
	}
	return nil
}

// add sign for url by secret
func (c *DingtalkNotify) addSign(webhookURL string, secret string) string {
	webhook := webhookURL
	if secret != "" {
		timestamp := time.Now().UnixMilli()
		stringToSign := fmt.Sprint(timestamp, "\n", secret)
		h := hmac.New(sha256.New, []byte(secret))
		h.Write([]byte(stringToSign))
		sign := url.QueryEscape(base64.StdEncoding.EncodeToString(h.Sum(nil)))
		webhook = fmt.Sprint(webhookURL, "&timestamp=", timestamp, "&sign="+sign)
	}
	logs.Debug("[%s / %s] - Dingtalk webhook: %s", c.Kind(), c.Name(), webhook)
	return webhook
}
