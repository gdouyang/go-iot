package email

import (
	"encoding/json"
	"go-iot/notify"
	"net"
	"strconv"
	"strings"

	"gopkg.in/gomail.v2"
)

func init() {
	notify.RegNotify(func() notify.Notify {
		return &NotifyConfig{}
	})
}

// NotifyConfig is the email notification configuration
type NotifyConfig struct {
	Server string `json:"server"`
	User   string `json:"username"`
	Pass   string `json:"password"`
	To     string `json:"to"`
	From   string `json:"from,omitempty"`
}

func (c *NotifyConfig) Kind() string {
	return "email"
}

func (c *NotifyConfig) Name() string {
	return "邮件"
}

func (c *NotifyConfig) FromJson(str string) error {
	err := json.Unmarshal([]byte(str), c)
	return err
}
func (c *NotifyConfig) Config() []map[string]string {
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
func (c *NotifyConfig) Notify(subject string, message string) error {

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
	m.SetHeader("Subject", subject)
	m.SetBody("text/html; charset=UTF-8", message)

	d := gomail.NewDialer(host, port, c.User, c.Pass)
	err = d.DialAndSend(m)

	return err
}
