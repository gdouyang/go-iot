package sender

import (
	"encoding/json"
	"go-iot/models"
	"go-iot/models/material"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/astaxie/beego"
)

const (
	MATERIAL_DOWNLOAD = "materialDownload"
)

type MaterialSender struct {
}

func (this MaterialSender) Download(data []byte) models.JsonResp {
	var m material.Material
	json.Unmarshal(data, &m)

	agent_server_ip := beego.AppConfig.String("agent_server_ip")

	res, err := http.Get("http://" + agent_server_ip + "/file/" + m.Id)
	if err != nil {
		return models.JsonResp{Success: false, Msg: err.Error()}
	}
	//执行close之前一定要判断错误，如没有body会崩溃
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	os.Mkdir("./file", os.ModePerm)
	err = ioutil.WriteFile("./"+m.Path, body, os.ModePerm)
	if err != nil {
		return models.JsonResp{Success: false, Msg: err.Error()}
	}
	return models.JsonResp{Success: true, Msg: ""}
}
