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
	Server string `yaml:"server" json:"server" jsonschema:"required,format=hostname,title=SMTP Server,description=SMTP server with port,example=\"smtp.example.com:465\""`
	User   string `yaml:"username" json:"username" jsonschema:"required,title=SMTP Username,description=SMTP username,example=\"name@example.com\""`
	Pass   string `yaml:"password" json:"password" jsonschema:"required,title=SMTP Password,description=SMTP password,example=\"password\""`
	To     string `yaml:"to" json:"to" jsonschema:"required,title=To,description=Email address to send,example=\"usera@example.com;userb@example.com\""`
	From   string `yaml:"from,omitempty" json:"from,omitempty" jsonschema:"title=From,description=Email address from,example=\"from@example.com\""`
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
