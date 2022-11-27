package email_test

import (
	"go-iot/notify"
	"go-iot/notify/email"
	"testing"

	"github.com/beego/beego/v2/core/logs"
	"github.com/stretchr/testify/assert"
)

func TestEmain(t *testing.T) {
	var data = map[string]interface{}{
		"name": "sss",
		"age":  1,
		"obj": map[string]string{
			"name": "test",
		},
	}
	e := email.EmailNotify{}
	config := notify.NotifyConfig{
		Config:   `{"server":"localhost","username":"test", "password":"123", "to":"abc@q.com"}`,
		Template: `{"subject":"Test Title", "text": "you have email name=${name} age=${age} obj.name=${obj.name}"}`,
	}
	err := e.FromJson(config)
	if err != nil {
		logs.Error(err)
	}
	result := e.ParseTemplate(data)
	logs.Info(result)
	assert.Equal(t, "you have email name=sss age=1 obj.name=test", result)
}
