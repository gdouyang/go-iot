package api

import (
	"errors"
	"go-iot/pkg/api/web"
	"go-iot/pkg/cluster"
	"go-iot/pkg/models"
	rule "go-iot/pkg/models/rule"
	"go-iot/pkg/ruleengine"
	"net/http"
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
	RegResource(sceneResource)

	var api *ruleApi

	web.RegisterAPI("/rule/page", "POST", api.page)
	// 新增规则
	web.RegisterAPI("/rule", "POST", api.add)
	// 修改规则
	web.RegisterAPI("/rule/{id}", "PUT", api.update)
	web.RegisterAPI("/rule/{id}", "GET", api.get)
	// 删除规则
	web.RegisterAPI("/rule/{id}", "DELETE", api.delete)
	// 启动规则
	web.RegisterAPI("/rule/{id}/start", "POST", api.start)
	// 停用规则
	web.RegisterAPI("/rule/{id}/stop", "POST", api.stop)
}

var ruleApi1 ruleApi

type ruleApi struct {
}

func (a *ruleApi) page(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
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
func (a *ruleApi) add(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(sceneResource, CretaeAction) {
		return
	}
	var ob models.RuleModel
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	executor := ruleModelToRuleExecutor(&ob)
	err = executor.Valid()
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
func (a *ruleApi) update(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(sceneResource, SaveAction) {
		return
	}
	var ob models.RuleModel
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	executor := ruleModelToRuleExecutor(&ob)
	err = executor.Valid()
	if err != nil {
		ctl.RespError(err)
		return
	}
	_, err = ruleApi1.getRuleAndCheckCreateId(ctl, ob.Id)
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
func (a *ruleApi) get(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(sceneResource, QueryAction) {
		return
	}
	id := ctl.Param("id")
	_id, err := strconv.Atoi(id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	u, err := ruleApi1.getRuleAndCheckCreateId(ctl, int64(_id))
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOkData(u)
}
func (a *ruleApi) delete(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(sceneResource, DeleteAction) {
		return
	}
	id := ctl.Param("id")
	_id, err := strconv.Atoi(id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	_, err = ruleApi1.getRuleAndCheckCreateId(ctl, int64(_id))
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
func (a *ruleApi) start(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(sceneResource, SaveAction) {
		return
	}
	ruleApi1.enable(ctl, true)
}
func (a *ruleApi) stop(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(sceneResource, SaveAction) {
		return
	}
	ruleApi1.enable(ctl, false)
}
func (a *ruleApi) enable(ctl *AuthController, flag bool) {
	id := ctl.Param("id")
	_id, err := strconv.Atoi(id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	m, err := ruleApi1.getRuleAndCheckCreateId(ctl, int64(_id))
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
func (a *ruleApi) getRuleAndCheckCreateId(ctl *AuthController, ruleId int64) (*models.RuleModel, error) {
	ob, err := rule.GetRuleMust(ruleId)
	if err != nil {
		return nil, err
	}
	if ob.CreateId != ctl.GetCurrentUser().Id {
		return nil, errors.New("data is not you created")
	}
	return ob, nil
}
