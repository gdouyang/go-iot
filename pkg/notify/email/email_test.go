package email_test

import (
	"go-iot/pkg/notify"
	"go-iot/pkg/notify/email"
	"testing"

	logs "go-iot/pkg/logger"

	"github.com/stretchr/testify/assert"
)

func TestEmailParseTemplate(t *testing.T) {
	logs.InitNop()
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
	assert.NoError(t, err)
	result := e.ParseTemplate(data)
	assert.Equal(t, "you have email name=sss age=1 obj.name=test", result)
}
