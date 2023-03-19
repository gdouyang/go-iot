package email

import (
	"bytes"
	"encoding/json"
	"go-iot/pkg/notify"
	"html/template"
	"net"
	"strconv"
	"strings"

	"github.com/beego/beego/v2/core/logs"
	"gopkg.in/gomail.v2"
)

func init() {
	notify.RegNotify(func() notify.Notify {
		return &EmailNotify{}
	})
}

// EmailNotify is the email notification configuration
type EmailNotify struct {
	Server      string             `json:"server"`
	User        string             `json:"username"`
	Pass        string             `json:"password"`
	To          string             `json:"to"`
	From        string             `json:"from,omitempty"`
	name        string             `json:"-"`
	subject     string             `json:"-"`
	msgTemplate string             `json:"-"`
	template    *template.Template `json:"-"`
}

func (c *EmailNotify) Kind() string {
	return "email"
}

func (c *EmailNotify) Name() string {
	return "邮箱"
}

func (c *EmailNotify) ParseTemplate(data map[string]interface{}) string {
	var result bytes.Buffer
	if err := c.template.Execute(&result, data); err != nil {
		logs.Error(err)
		return c.msgTemplate
	}
	return result.String()
}

func (c *EmailNotify) FromJson(config notify.NotifyConfig) error {
	err := json.Unmarshal([]byte(config.Config), c)
	if err != nil {
		return err
	}
	c.name = config.Name
	tpl := map[string]string{}
	err = json.Unmarshal([]byte(config.Template), &tpl)
	c.subject = tpl["subject"]
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

func (c *EmailNotify) Meta() []map[string]string {
	var m []map[string]string = []map[string]string{
		{"name": "server", "type": "string", "required": "true", "title": "SMTP Server", "desc": "SMTP server with port,example=\"smtp.example.com:465\""},
		{"name": "username", "type": "string", "required": "true", "title": "SMTP Username", "desc": "SMTP username,example=\"name@example.com\""},
		{"name": "password", "type": "password", "required": "true", "title": "SMTP Password", "desc": "SMTP password"},
		{"name": "from", "type": "string", "title": "From", "desc": "Email address from,example=\"from@example.com\""},
		{"name": "to", "type": "string", "required": "true", "title": "To", "desc": "Email address to send,example=\"usera@example.com;userb@example.com\""},
	}
	return m
}

// SendMail sends the email
func (c *EmailNotify) Notify(message string) error {

	host, p, err := net.SplitHostPort(c.Server)
	if err != nil {
		return err
	}

	port, err := strconv.Atoi(p)
	if err != nil {
		return err
	}

	email := "Notification" + "<" + c.User + ">"
	if c.From != "" {
		email = c.From
	}

	split := func(r rune) bool {
		return r == ';' || r == ','
	}

	recipients := strings.FieldsFunc(c.To, split)

	m := gomail.NewMessage()
	m.SetHeader("From", email)
	m.SetHeader("To", recipients...)
	m.SetHeader("Subject", c.subject)
	m.SetBody("text/html; charset=UTF-8", message)

	d := gomail.NewDialer(host, port, c.User, c.Pass)
	err = d.DialAndSend(m)

	return err
}
