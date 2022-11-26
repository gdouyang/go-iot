package email_test

import (
	"bytes"
	"html/template"
	"log"
	"testing"
)

func TestEmain(t *testing.T) {
	tpl := template.New("").Delims("${", "}")
	template := template.Must(tpl.Parse("this is a email test name=${.name} age=${.age} obj.name=${.obj.name}"))
	var data = map[string]interface{}{
		"name": "sss",
		"age":  1,
		"obj": map[string]string{
			"name": "test",
		},
	}
	var result bytes.Buffer
	if err := template.Execute(&result, data); err != nil {
		log.Fatalln(err)
	}
	log.Println(result.String())
}
