package api

import (
	"errors"
	"go-iot/pkg/api/web"
	"go-iot/pkg/cluster"
	"go-iot/pkg/models"
	rule "go-iot/pkg/models/rule"
	"go-iot/pkg/ruleengine"
	"strconv"
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
	web.RegisterAPI("/rule/page", "POST", &RuleController{}, "Page")
	web.RegisterAPI("/rule", "POST", &RuleController{}, "Add")
	web.RegisterAPI("/rule/{id}", "PUT", &RuleController{}, "Update")
	web.RegisterAPI("/rule/{id}", "GET", &RuleController{}, "Get")
	web.RegisterAPI("/rule/{id}", "DELETE", &RuleController{}, "Delete")
	web.RegisterAPI("/rule/{id}/start", "POST", &RuleController{}, "Enable")
	web.RegisterAPI("/rule/{id}/stop", "POST", &RuleController{}, "Disable")

	RegResource(sceneResource)
}

type RuleController struct {
	AuthController
}

func (ctl *RuleController) Page() {
	if ctl.isForbidden(sceneResource, QueryAction) {
		return
	}
	var ob models.PageQuery
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}

	res, err := rule.PageRule(&ob, &ctl.GetCurrentUser().Id)
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
	id := ctl.Param("id")
	_id, err := strconv.Atoi(id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	u, err := ctl.getRuleAndCheckCreateId(int64(_id))
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
	ob.CreateId = ctl.GetCurrentUser().Id
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
	_, err = ctl.getRuleAndCheckCreateId(ob.Id)
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
	id := ctl.Param("id")
	_id, err := strconv.Atoi(id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	_, err = ctl.getRuleAndCheckCreateId(int64(_id))
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
	id := ctl.Param("id")
	_id, err := strconv.Atoi(id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	m, err := ctl.getRuleAndCheckCreateId(int64(_id))
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
	if ctl.IsNotClusterRequest() {
		err = rule.UpdateRuleStatus(state, m.Id)
		if err != nil {
			ctl.RespError(err)
			return
		}
		cluster.BroadcastInvoke(ctl.Request)
	}
	ctl.RespOk()
}

func (ctl *RuleController) getRuleAndCheckCreateId(ruleId int64) (*models.RuleModel, error) {
	ob, err := rule.GetRuleMust(ruleId)
	if err != nil {
		return nil, err
	}
	if ob.CreateId != ctl.GetCurrentUser().Id {
		return nil, errors.New("data is not you created")
	}
	return ob, nil
}
