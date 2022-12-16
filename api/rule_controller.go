package api

import (
	"go-iot/models"
	rule "go-iot/models/rule"
	"go-iot/ruleengine"
	"strconv"

	"github.com/beego/beego/v2/server/web"
)

var sceneResource = Resource{
	Id:   "rule-mgr",
	Name: "规则引擎",
	Action: []ResourceAction{
		QueryAction,
		CretaeAction,
		SaveAction,
		DeleteAction,
	},
}

func init() {
	ns := web.NewNamespace("/api/rule",
		web.NSRouter("/page", &RuleController{}, "post:List"),
		web.NSRouter("/", &RuleController{}, "post:Add"),
		web.NSRouter("/:id", &RuleController{}, "put:Update"),
		web.NSRouter("/:id", &RuleController{}, "get:Get"),
		web.NSRouter("/:id", &RuleController{}, "delete:Delete"),
		web.NSRouter("/:id/start", &RuleController{}, "post:Enable"),
		web.NSRouter("/:id/stop", &RuleController{}, "post:Disable"),
	)
	web.AddNamespace(ns)

	regResource(sceneResource)
}

type RuleController struct {
	AuthController
}

func (ctl *RuleController) List() {
	if ctl.isForbidden(sceneResource, QueryAction) {
		return
	}
	var ob models.PageQuery
	ctl.BindJSON(&ob)

	res, err := rule.PageRule(&ob)
	if err != nil {
		ctl.RespError(err)
	} else {
		ctl.RespOkData(res)
	}
}

func (ctl *RuleController) Get() {
	if ctl.isForbidden(sceneResource, QueryAction) {
		return
	}
	id := ctl.Param(":id")
	_id, err := strconv.Atoi(id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	u, err := rule.GetRuleMust(int64(_id))
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOkData(u)
}

func (ctl *RuleController) Add() {
	if ctl.isForbidden(sceneResource, CretaeAction) {
		return
	}
	var ob models.RuleModel
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	err = rule.AddRule(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

func (ctl *RuleController) Update() {
	if ctl.isForbidden(sceneResource, SaveAction) {
		return
	}
	var ob models.RuleModel
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	err = rule.UpdateRule(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

func (ctl *RuleController) Delete() {
	if ctl.isForbidden(sceneResource, DeleteAction) {
		return
	}
	id := ctl.Param(":id")
	_id, err := strconv.Atoi(id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	var ob *models.Rule = &models.Rule{
		Id: int64(_id),
	}
	err = rule.DeleteRule(ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

func (ctl *RuleController) Enable() {
	if ctl.isForbidden(sceneResource, SaveAction) {
		return
	}
	ctl.enable(true)
}

func (ctl *RuleController) Disable() {
	if ctl.isForbidden(sceneResource, SaveAction) {
		return
	}
	ctl.enable(false)
}

func (ctl *RuleController) enable(flag bool) {
	id := ctl.Param(":id")
	_id, err := strconv.Atoi(id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	m, err := rule.GetRuleMust(int64(_id))
	if err != nil {
		ctl.RespError(err)
		return
	}
	var state string = models.Started
	if flag {
		rule := ruleModelToRuleExecutor(m)
		err = ruleengine.Start(m.Id, &rule)
		if err != nil {
			ctl.RespError(err)
			return
		}
	} else {
		state = models.Stopped
		ruleengine.Stop(m.Id)
	}

	err = rule.UpdateRuleStatus(state, m.Id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

func ruleModelToRuleExecutor(m *models.RuleModel) ruleengine.RuleExecutor {
	rule := ruleengine.RuleExecutor{
		Name:        m.Name,
		Type:        m.Type,
		ProductId:   m.ProductId,
		TriggerType: ruleengine.TriggerType(m.TriggerType),
		Cron:        m.Cron,
		Trigger:     m.Trigger,
		Actions:     m.Actions,
		DeviceIds:   m.DeviceIds,
	}
	return rule
}
