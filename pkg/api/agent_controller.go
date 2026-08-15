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
)

var agentResource = Resource{
	Id:     "agent-mgr",
	Name:   "AI 助手",
	Sort:   15,
	Action: []ResourceAction{QueryAction, CretaeAction, SaveAction, DeleteAction},
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
	if ctl.GetCurrentUser() == nil {
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
	if ctl.GetCurrentUser() == nil {
		return
	}
	st, err := agent.DefaultStore.GetSettings(ctl.GetCurrentUser().Id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOkData(agent.ViewSettings(st))
}

func (a *agentApi) putSettings(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.GetCurrentUser() == nil {
		return
	}
	var put agent.SettingsPut
	if err := ctl.BindJSON(&put); err != nil {
		a.respReason(ctl, http.StatusBadRequest, agent.ReasonInvalidArgs, err.Error())
		return
	}
	st, err := agent.PutSettings(agent.DefaultStore, ctl.GetCurrentUser().Id, put)
	if err != nil {
		a.respReason(ctl, http.StatusBadRequest, agent.ReasonInvalidArgs, err.Error())
		return
	}
	ctl.RespOkData(agent.ViewSettings(st))
}

func (a *agentApi) createConv(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(agentResource, CretaeAction) {
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
	ctl.RespOkData(c)
}

func (a *agentApi) pageConv(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(agentResource, QueryAction) {
		return
	}
	list, err := agent.DefaultStore.ListConversations(ctl.GetCurrentUser().Id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOkData(models.PageUtil(int64(len(list)), 1, 50, list))
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
	drafts, _ := agent.DefaultStore.ListDrafts(c.Id)
	var pending []map[string]any
	for _, d := range agent.PendingDrafts(drafts) {
		pending = append(pending, map[string]any{
			"id": d.Id, "tool": d.ToolName, "preview": json.RawMessage(orEmptyJSON(d.Preview)),
			"expireTime": d.ExpireTime, "draftStatus": d.Status,
		})
	}
	ctl.RespOkData(map[string]any{
		"conversation": c,
		"runStatus":    c.RunStatus,
		"needsResume":  c.NeedsResume,
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
	c.Title = title
	c.UpdateTime = models.NewDateTime()
	if err := agent.DefaultStore.SaveConversation(c); err != nil {
		ctl.RespError(err)
		return
	}
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
	for _, m := range tail {
		dto := map[string]any{
			"id": m.Id, "role": m.Role, "content": m.Content,
			"eventType": m.EventType, "toolName": m.ToolName,
			"toolCallId": m.ToolCallId, "draftId": m.DraftId,
			"productId": m.ProductId, "createdTime": m.CreateTime, "createTimeMs": m.CreateTimeMs,
		}
		if m.EventType == agent.EventConfirmRequired {
			dto["preview"] = json.RawMessage(orEmptyJSON(extractPreview(m.Payload)))
			dto["draftStatus"] = draftStatusOf(c.Id, m.DraftId)
		}
		dtos = append(dtos, dto)
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
	if err := ctl.BindJSON(&body); err != nil || strings.TrimSpace(body.Content) == "" || len(body.Content) > 16384 {
		a.respReason(ctl, http.StatusBadRequest, agent.ReasonInvalidArgs, "invalid content")
		return
	}
	res, err := agent.RunTurn(r.Context(), agent.DefaultStore, c, st, body.Content, false)
	if err != nil {
		a.mapRunErr(ctl, err)
		return
	}
	ctl.RespOkData(res)
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
	if c.RunStatus != agent.RunIdle || !c.NeedsResume {
		a.respReason(ctl, http.StatusConflict, "needsResume=false", "cannot continue")
		return
	}
	res, err := agent.RunTurn(r.Context(), agent.DefaultStore, c, st, "", true)
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
	if !agent.TryLock(c.Id) {
		a.respReason(ctl, http.StatusConflict, agent.ReasonRunActive, "busy")
		return
	}
	res, err := agent.ApplyDrafts(agent.DefaultStore, ctl.GetCurrentUser().Id, c, body.DraftIds, body.Resume)
	agent.Unlock(c.Id)
	if err != nil {
		a.mapRunErr(ctl, err)
		return
	}
	if body.Resume && res.NeedsResume {
		st, _ := agent.DefaultStore.GetSettings(ctl.GetCurrentUser().Id)
		if agent.ModelConfigured(st) {
			if c2, err2 := a.mustOwnConv(ctl, c.Id); err2 == nil {
				run, runErr := agent.RunTurn(r.Context(), agent.DefaultStore, c2, st, "", true)
				if runErr == nil && run != nil {
					res.NeedsResume = run.NeedsResume
					res.RunStatus = run.RunStatus
					res.PendingDrafts = run.PendingDrafts
				}
			}
		}
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
	if !agent.TryLock(c.Id) {
		a.respReason(ctl, http.StatusConflict, agent.ReasonRunActive, "busy")
		return
	}
	defer agent.Unlock(c.Id)
	res, err := agent.RejectDrafts(agent.DefaultStore, ctl.GetCurrentUser().Id, c, body.DraftIds)
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
	if c.RunStatus == agent.RunApplying {
		ctl.RespOk()
		return
	}
	if agent.PendingLeft(agent.DefaultStore, c.Id) {
		c.RunStatus = agent.RunAwaitingConfirm
	} else {
		c.RunStatus = agent.RunIdle
	}
	_ = agent.DefaultStore.SaveConversation(c)
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
	case strings.Contains(msg, "llm http"):
		a.respReason(ctl, http.StatusBadGateway, "llm_error", msg)
	default:
		ctl.RespError(err)
	}
}

func orEmptyJSON(s string) string {
	if strings.TrimSpace(s) == "" {
		return "null"
	}
	return s
}

func extractPreview(payload string) string {
	var m map[string]json.RawMessage
	if json.Unmarshal([]byte(payload), &m) != nil {
		return payload
	}
	if p, ok := m["preview"]; ok {
		return string(p)
	}
	return payload
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


