package controllers

import (
	"encoding/json"
	"go-iot/models"
	"go-iot/models/camera"
	"go-iot/server"

	"github.com/astaxie/beego"
	"github.com/gwuhaolin/livego/configure"
)

var liveServer *server.LiveMedia

// 流媒体服务
func init() {
	// 初始启动
	livecfg := new(configure.Application)
	livecfg.Appname = "live"
	livecfg.Liveon = "on"
	livecfg.Hlson = "on"
	configure.RtmpServercfg.Server = []configure.Application{*livecfg}
	liveServer = server.NEW()
	liveServer.ResumeAll()
	//健康检查
	beego.Router("/mediasrs/list", &MediaServerController{}, "post:List")
	beego.Router("/mediasrs/startall", &MediaServerController{}, "put:StartAll")
	beego.Router("/mediasrs/stopall", &MediaServerController{}, "put:StopAll")
	beego.Router("/mediasrs/start/:id", &MediaServerController{}, "post:Start")
	beego.Router("/mediasrs/stop/:id", &MediaServerController{}, "post:Stop")
}

type MediaServerController struct {
	beego.Controller
}

// 查询设备列表
func (this *MediaServerController) List() {
	ob := models.PageQuery{}
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)
	res, err := camera.ListMediaServer(&ob)
	if err != nil {
		this.Data["json"] = models.JsonResp{Success: false, Msg: err.Error()}
	} else {

		this.Data["json"] = &res
	}
	this.ServeJSON()
}

func (this *MediaServerController) StartAll() {
	liveServer.Start("all")
	this.Data["json"] = models.JsonResp{Success: true, Msg: "操作完成，稍候查看状态"}
	this.ServeJSON()
}

func (this *MediaServerController) StopAll() {
	err := liveServer.StopAll()
	if err != nil {
		this.Data["json"] = models.JsonResp{Success: false, Msg: err.Error()}
		this.ServeJSON()
	}
	this.Data["json"] = models.JsonResp{Success: true, Msg: "停止成功"}
	this.ServeJSON()
}

func (this *MediaServerController) Start() {
	beego.Info("start 1")
}

func (this *MediaServerController) Stop() {
	beego.Info("stop 1")
}

func (this *MediaServerController) Update() {
	//do stop server before update
	beego.Info("say something here")
}
