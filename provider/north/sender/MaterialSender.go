package sender

import (
	"encoding/json"
	"go-iot/agent"
	"go-iot/models"
	"go-iot/models/material"
	"go-iot/provider/util"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/astaxie/beego"
)

const (
	MATERIAL_DOWNLOAD = "materialDownload"
)

func init() {
	mSender := MaterialSender{}
	agent.RegProcessFunc(MATERIAL_DOWNLOAD, func(request agent.AgentRequest) models.JsonResp {
		res := mSender.Download(request.Data)
		return res
	})
}

type MaterialSender struct {
}

func (this MaterialSender) Download(iotReq models.IotRequest) models.JsonResp {
	data := iotReq.Data
	var m material.Material
	json.Unmarshal(data, &m)

	agent_server_ip := beego.AppConfig.String("agent_server_ip")

	res, err := http.Get("http://" + agent_server_ip + "/file/" + m.Path)
	if err != nil {
		return models.JsonResp{Success: false, Msg: err.Error()}
	}
	if res.StatusCode != 200 {
		return models.JsonResp{Success: false, Msg: res.Status}
	}
	//执行close之前一定要判断错误，如没有body会崩溃
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	os.Mkdir("./files", os.ModePerm)
	path := "./files/" + m.Path
	exist, _ := util.FileExists(path)
	if exist {
		fileSize := util.FileSize(path)
		length, err := strconv.ParseInt(m.Size, 10, 64)
		if err != nil {
			return models.JsonResp{Success: false, Msg: err.Error()}
		}
		if fileSize != length {
			exist = false
		}
	}
	if !exist {
		err = ioutil.WriteFile(path, body, os.ModePerm)
		if err != nil {
			return models.JsonResp{Success: false, Msg: err.Error()}
		}
	} else {
		beego.Info(path, " 已存在本地")
	}
	return models.JsonResp{Success: true, Msg: ""}
}
