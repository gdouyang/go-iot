package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"go-iot/pkg/agent"
	"go-iot/pkg/api/web"
	"go-iot/pkg/common"
	"go-iot/pkg/models"
	"time"
)

var agentResource = Resource{
	Id:     "agent-mgr",
	Name:   "AI 助手",
	Sort:   15,
	Action: []ResourceAction{QueryAction, CreateAction, SaveAction, DeleteAction},
}

func init() {
	RegResource(agentResource)
	api := &agentApi{}
	web.RegisterAPI("/agent/status", "GET", api.status)
	web.RegisterAPI("/agent/settings", "GET", api.getSettings)
	web.RegisterAPI("/agent/settings", "PUT", api.putSettings)
	web.RegisterAPI("/agent/conversations", "POST", api.createConv)
	web.RegisterAPI("/agent/conversations/page", "POST", api.pageConv)
	web.RegisterAPI("/agent/conversations/{id}", "GET", api.getConv)
	web.RegisterAPI("/agent/conversations/{id}", "PUT", api.putConv)
	web.RegisterAPI("/agent/conversations/{id}", "DELETE", api.deleteConv)
	web.RegisterAPI("/agent/conversations/{id}/messages", "GET", api.listMessages)
	web.RegisterAPI("/agent/conversations/{id}/messages", "POST", api.postMessage)
	web.RegisterAPI("/agent/conversations/{id}/continue", "POST", api.continueRun)
	web.RegisterAPI("/agent/conversations/{id}/apply", "POST", api.apply)
	web.RegisterAPI("/agent/conversations/{id}/reject", "POST", api.reject)
	web.RegisterAPI("/agent/conversations/{id}/cancel", "POST", api.cancel)
}

type agentApi struct{}

func (a *agentApi) respReason(ctl *AuthController, status int, reason, msg string) {
	ctl.JSON(common.JsonResp{
		Success: false,
		Code:    status,
		Msg:     msg,
		Result:  map[string]string{"reason": reason},
	})
}

func (a *agentApi) mustOwnConv(ctl *AuthController, id string) (*models.AgentConversation, error) {
	c, err := agent.DefaultStore.GetConversation(id)
	if err != nil || c == nil || c.CreateId != ctl.GetCurrentUser().Id {
		return nil, errors.New(agent.ReasonNotFound)
	}
	return c, nil
}

func (a *agentApi) status(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(agentResource, QueryAction) {
		return
	}
	st, _ := agent.DefaultStore.GetSettings(ctl.GetCurrentUser().Id)
	reason := ""
	if !agent.ModelConfigured(st) {
		reason = agent.ReasonModelNotConfigured
	}
	ctl.RespOkData(map[string]any{
		"modelConfigured": agent.ModelConfigured(st),
		"reason":          reason,
		"defaultBaseUrl":  agent.DefaultBaseURL,
		"defaultModel":    agent.DefaultModel,
	})
}

func (a *agentApi) getSettings(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(agentResource, QueryAction) {
		return
	}
	st, err := agent.DefaultStore.GetSettings(ctl.GetCurrentUser().Id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	v := agent.ViewSettings(st)
	u := ctl.GetCurrentUser()
	v.CanAllowPrivateLLM = agent.IsPlatformAdmin(u.Id, u.Username)
	ctl.RespOkData(v)
}

func (a *agentApi) putSettings(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(agentResource, SaveAction) {
		return
	}
	var put agent.SettingsPut
	if err := ctl.BindJSON(&put); err != nil {
		a.respReason(ctl, http.StatusBadRequest, agent.ReasonInvalidArgs, err.Error())
		return
	}
	u := ctl.GetCurrentUser()
	if put.AllowPrivateLLM != nil && *put.AllowPrivateLLM && !agent.IsPlatformAdmin(u.Id, u.Username) {
		a.respReason(ctl, http.StatusForbidden, agent.ReasonForbidden, "only admin can allow private llm")
		return
	}
	st, err := agent.PutSettings(agent.DefaultStore, u.Id, put)
	if err != nil {
		if strings.Contains(err.Error(), agent.ReasonForbidden) {
			a.respReason(ctl, http.StatusForbidden, agent.ReasonForbidden, err.Error())
			return
		}
		a.respReason(ctl, http.StatusBadRequest, agent.ReasonInvalidArgs, err.Error())
		return
	}
	v := agent.ViewSettings(st)
	v.CanAllowPrivateLLM = agent.IsPlatformAdmin(u.Id, u.Username)
	ctl.RespOkData(v)
}

func (a *agentApi) createConv(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(agentResource, CreateAction) {
		return
	}
	var body struct {
		Title     string `json:"title"`
		ProductId string `json:"productId"`
	}
	_ = ctl.BindJSON(&body)
	now := models.NewDateTime()
	c := &models.AgentConversation{
		Id: agent.NewHexID(), Title: body.Title, ProductId: body.ProductId,
		Status: "active", RunStatus: agent.RunIdle,
		CreateId: ctl.GetCurrentUser().Id, CreateTime: now, UpdateTime: now,
	}
	if c.Title == "" {
		c.Title = agent.DefaultConvTitle
	}
	if err := agent.DefaultStore.SaveConversation(c); err != nil {
		ctl.RespError(err)
		return
	}
	agent.EnsureConvSeq(r.Context(), c.Id, agent.DefaultStore)
	ctl.RespOkData(c)
}

func (a *agentApi) pageConv(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(agentResource, QueryAction) {
		return
	}
	var q models.PageQuery
	_ = ctl.BindJSON(&q)
	if q.PageNum <= 0 {
		q.PageNum = 1
	}
	if q.PageSize <= 0 || q.PageSize > 100 {
		q.PageSize = 50
	}
	list, err := agent.DefaultStore.ListConversations(ctl.GetCurrentUser().Id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	total := int64(len(list))
	start := (q.PageNum - 1) * q.PageSize
	if start > len(list) {
		start = len(list)
	}
	end := start + q.PageSize
	if end > len(list) {
		end = len(list)
	}
	ctl.RespOkData(models.PageUtil(total, q.PageNum, q.PageSize, list[start:end]))
}

func (a *agentApi) getConv(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(agentResource, QueryAction) {
		return
	}
	c, err := a.mustOwnConv(ctl, ctl.Param("id"))
	if err != nil {
		a.respReason(ctl, http.StatusNotFound, agent.ReasonNotFound, "not found")
		return
	}
	agent.SyncRunStatus(agent.DefaultStore, c)
	drafts, _ := agent.DefaultStore.ListDrafts(c.Id)
	var pending []map[string]any
	for _, d := range agent.PendingDrafts(drafts) {
		pending = append(pending, map[string]any{
			"id": d.Id, "tool": d.ToolName, "preview": json.RawMessage(orEmptyJSON(d.Preview)),
			"expireTime": d.ExpireTime, "draftStatus": d.Status,
		})
	}
	ctl.RespOkData(map[string]any{
		"conversation":  c,
		"runStatus":     c.RunStatus,
		"needsResume":   c.NeedsResume,
		"pendingDrafts": pending,
	})
}

func (a *agentApi) putConv(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(agentResource, SaveAction) {
		return
	}
	c, err := a.mustOwnConv(ctl, ctl.Param("id"))
	if err != nil {
		a.respReason(ctl, http.StatusNotFound, agent.ReasonNotFound, "not found")
		return
	}
	var body struct {
		Title string `json:"title"`
	}
	if err := ctl.BindJSON(&body); err != nil {
		a.respReason(ctl, http.StatusBadRequest, agent.ReasonInvalidArgs, err.Error())
		return
	}
	title, err := agent.NormalizeUserTitle(body.Title)
	if err != nil {
		a.respReason(ctl, http.StatusBadRequest, agent.ReasonInvalidArgs, err.Error())
		return
	}
	if err := agent.DefaultStore.UpdateTitle(c.Id, title); err != nil {
		ctl.RespError(err)
		return
	}
	c.Title = title
	ctl.RespOkData(c)
}

func (a *agentApi) deleteConv(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(agentResource, DeleteAction) {
		return
	}
	c, err := a.mustOwnConv(ctl, ctl.Param("id"))
	if err != nil {
		a.respReason(ctl, http.StatusNotFound, agent.ReasonNotFound, "not found")
		return
	}
	_ = agent.DefaultStore.DeleteConversation(c.Id)
	ctl.RespOk()
}

func (a *agentApi) listMessages(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(agentResource, QueryAction) {
		return
	}
	c, err := a.mustOwnConv(ctl, ctl.Param("id"))
	if err != nil {
		a.respReason(ctl, http.StatusNotFound, agent.ReasonNotFound, "not found")
		return
	}
	msgs, err := agent.DefaultStore.ListMessages(c.Id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	pageSize := 50
	tail := agent.TailMessages(msgs, pageSize)
	dtos := make([]map[string]any, 0, len(tail))
	for i := range tail {
		st := ""
		if tail[i].EventType == agent.EventConfirmRequired {
			st = draftStatusOf(c.Id, tail[i].DraftId)
		}
		dtos = append(dtos, agent.MessageView(&tail[i], st))
	}
	ctl.RespOkData(models.PageUtil(int64(len(msgs)), 1, pageSize, dtos))
}

func (a *agentApi) requireModel(ctl *AuthController) (*models.AgentUserSettings, bool) {
	st, _ := agent.DefaultStore.GetSettings(ctl.GetCurrentUser().Id)
	if !agent.ModelConfigured(st) {
		a.respReason(ctl, http.StatusConflict, agent.ReasonModelNotConfigured, "configure model first")
		return nil, false
	}
	return st, true
}

const maxMessageContentLength = 131072 // 128 KB, supports attached hardware protocol docs up to 64KB

func (a *agentApi) postMessage(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(agentResource, SaveAction) {
		return
	}
	st, ok := a.requireModel(ctl)
	if !ok {
		return
	}
	c, err := a.mustOwnConv(ctl, ctl.Param("id"))
	if err != nil {
		a.respReason(ctl, http.StatusNotFound, agent.ReasonNotFound, "not found")
		return
	}
	agent.SyncRunStatus(agent.DefaultStore, c)
	if c.RunStatus == agent.RunRunning || c.RunStatus == agent.RunApplying {
		a.respReason(ctl, http.StatusConflict, agent.ReasonRunActive, "run active")
		return
	}
	if c.RunStatus == agent.RunAwaitingConfirm {
		a.respReason(ctl, http.StatusConflict, agent.ReasonPendingDrafts, "pending drafts")
		return
	}
	var body struct {
		Content string `json:"content"`
	}
	if err := ctl.BindJSON(&body); err != nil || strings.TrimSpace(body.Content) == "" || len(body.Content) > maxMessageContentLength {
		a.respReason(ctl, http.StatusBadRequest, agent.ReasonInvalidArgs, "invalid content")
		return
	}
	a.run(ctl, r, c, st, body.Content, false)
}

func (a *agentApi) continueRun(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(agentResource, SaveAction) {
		return
	}
	st, ok := a.requireModel(ctl)
	if !ok {
		return
	}
	c, err := a.mustOwnConv(ctl, ctl.Param("id"))
	if err != nil {
		a.respReason(ctl, http.StatusNotFound, agent.ReasonNotFound, "not found")
		return
	}
	agent.SyncRunStatus(agent.DefaultStore, c)
	if c.RunStatus != agent.RunIdle || !c.NeedsResume {
		a.respReason(ctl, http.StatusConflict, "needsResume=false", "cannot continue")
		return
	}
	a.run(ctl, r, c, st, "", true)
}

func (a *agentApi) toolCtx(ctl *AuthController) agent.ToolContext {
	perms := map[string]bool{}
	if s := ctl.GetSession(); s != nil {
		perms = s.GetPermission()
	}
	return agent.ToolContext{UserId: ctl.GetCurrentUser().Id, Perms: perms}
}

func (a *agentApi) run(ctl *AuthController, r *http.Request, c *models.AgentConversation, st *models.AgentUserSettings, text string, continueRun bool) {
	opts := agent.RunOpts{
		Store: agent.DefaultStore, Conv: c, Settings: st,
		UserText: text, Continue: continueRun, Perms: a.toolCtx(ctl).Perms,
	}
	if agent.WantsStream(r) {
		sink := agent.NewSSESink(ctl.ResponseWriter)
		if sink == nil {
			a.respReason(ctl, http.StatusInternalServerError, agent.ReasonInvalidArgs, "stream unsupported")
			return
		}
		opts.Sink = sink
		res, err := agent.RunTurnOpts(r.Context(), opts)
		if err != nil {
			sink.Emit("error", map[string]any{"message": err.Error(), "reason": runReason(err)})
			if res != nil {
				sink.Emit("done", res)
			}
			return
		}
		return
	}
	res, err := agent.RunTurnOpts(r.Context(), opts)
	if err != nil {
		a.mapRunErr(ctl, err)
		return
	}
	ctl.RespOkData(res)
}

func (a *agentApi) apply(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(agentResource, SaveAction) {
		return
	}
	c, err := a.mustOwnConv(ctl, ctl.Param("id"))
	if err != nil {
		a.respReason(ctl, http.StatusNotFound, agent.ReasonNotFound, "not found")
		return
	}
	var body struct {
		DraftIds []string `json:"draftIds"`
		Resume   bool     `json:"resume"`
	}
	if err := ctl.BindJSON(&body); err != nil {
		a.respReason(ctl, http.StatusBadRequest, agent.ReasonInvalidArgs, err.Error())
		return
	}
	agent.SyncRunStatus(agent.DefaultStore, c)
	if !agent.TryLock(c.Id) {
		a.respReason(ctl, http.StatusConflict, agent.ReasonRunActive, "busy")
		return
	}
	res, err := agent.ApplyDrafts(agent.DefaultStore, a.toolCtx(ctl), c, body.DraftIds, body.Resume)
	agent.Unlock(c.Id)
	if err != nil {
		a.mapRunErr(ctl, err)
		return
	}
	ctl.RespOkData(res)
}

func (a *agentApi) reject(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(agentResource, SaveAction) {
		return
	}
	c, err := a.mustOwnConv(ctl, ctl.Param("id"))
	if err != nil {
		a.respReason(ctl, http.StatusNotFound, agent.ReasonNotFound, "not found")
		return
	}
	var body struct {
		DraftIds []string `json:"draftIds"`
	}
	_ = ctl.BindJSON(&body)
	agent.SyncRunStatus(agent.DefaultStore, c)
	if !agent.TryLock(c.Id) {
		a.respReason(ctl, http.StatusConflict, agent.ReasonRunActive, "busy")
		return
	}
	defer agent.Unlock(c.Id)
	res, err := agent.RejectDrafts(agent.DefaultStore, a.toolCtx(ctl), c, body.DraftIds)
	if err != nil {
		a.mapRunErr(ctl, err)
		return
	}
	ctl.RespOkData(res)
}

func (a *agentApi) cancel(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(agentResource, SaveAction) {
		return
	}
	c, err := a.mustOwnConv(ctl, ctl.Param("id"))
	if err != nil {
		a.respReason(ctl, http.StatusNotFound, agent.ReasonNotFound, "not found")
		return
	}
	localFound := agent.CancelActive(c.Id)
	agent.BroadcastCancel(c.Id)
	if localFound {
		agent.WaitRun(c.Id, 3*time.Second)
	}
	c2, err := a.mustOwnConv(ctl, c.Id)
	if err != nil {
		a.respReason(ctl, http.StatusNotFound, agent.ReasonNotFound, "not found")
		return
	}
	if agent.IsLocked(c2.Id) && c2.RunStatus == agent.RunApplying {
		a.respReason(ctl, http.StatusConflict, agent.ReasonRunActive, "applying")
		return
	}
	if !agent.IsLocked(c2.Id) {
		if agent.PendingLeft(agent.DefaultStore, c2.Id) {
			c2.RunStatus = agent.RunAwaitingConfirm
		} else {
			c2.RunStatus = agent.RunIdle
		}
		c2.NeedsResume = false
		_ = agent.DefaultStore.SaveConversation(c2)
	}
	ctl.RespOk()
}

func (a *agentApi) mapRunErr(ctl *AuthController, err error) {
	msg := err.Error()
	switch {
	case strings.Contains(msg, agent.ReasonRunActive):
		a.respReason(ctl, http.StatusConflict, agent.ReasonRunActive, msg)
	case strings.Contains(msg, agent.ReasonPendingDrafts):
		a.respReason(ctl, http.StatusConflict, agent.ReasonPendingDrafts, msg)
	case strings.Contains(msg, agent.ReasonNotAwaiting):
		a.respReason(ctl, http.StatusConflict, agent.ReasonNotAwaiting, msg)
	case strings.Contains(msg, agent.ReasonToolsNotSupported):
		a.respReason(ctl, http.StatusBadRequest, agent.ReasonToolsNotSupported, msg)
	case strings.Contains(msg, agent.ReasonNotFound):
		a.respReason(ctl, http.StatusNotFound, agent.ReasonNotFound, msg)
	case strings.Contains(msg, agent.ReasonForbidden):
		a.respReason(ctl, http.StatusForbidden, agent.ReasonForbidden, msg)
	case strings.Contains(msg, "llm http"):
		a.respReason(ctl, http.StatusBadGateway, "llm_error", msg)
	default:
		ctl.RespError(err)
	}
}

func runReason(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	for _, r := range []string{
		agent.ReasonRunActive, agent.ReasonPendingDrafts, agent.ReasonNotAwaiting,
		agent.ReasonToolsNotSupported, agent.ReasonNotFound, agent.ReasonForbidden,
	} {
		if strings.Contains(msg, r) {
			return r
		}
	}
	if strings.Contains(msg, "llm http") {
		return "llm_error"
	}
	return "error"
}

func orEmptyJSON(s string) string {
	if strings.TrimSpace(s) == "" {
		return "null"
	}
	return s
}

func draftStatusOf(convId, draftId string) string {
	if draftId == "" {
		return ""
	}
	d, _ := agent.DefaultStore.GetDraft(draftId)
	if d == nil {
		return ""
	}
	return d.Status
}
