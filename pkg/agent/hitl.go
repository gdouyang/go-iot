package agent

import (
	"encoding/json"
	"fmt"
	"time"

	"go-iot/pkg/models"
)

var applyMutatingFn = ApplyMutating

type ApplyResult struct {
	Applied      []map[string]string `json:"applied"`
	Failed       []map[string]string `json:"failed"`
	RunStatus    string              `json:"runStatus"`
	NeedsResume  bool                `json:"needsResume"`
	PendingDrafts []string           `json:"pendingDrafts"`
}

func ApplyDrafts(store Store, userId int64, conv *models.AgentConversation, draftIds []string, resume bool) (*ApplyResult, error) {
	if conv.RunStatus == RunRunning || conv.RunStatus == RunApplying {
		return nil, fmt.Errorf("%s", ReasonRunActive)
	}
	if conv.RunStatus != RunAwaitingConfirm {
		return nil, fmt.Errorf("%s", ReasonNotAwaiting)
	}
	conv.RunStatus = RunApplying
	_ = store.SaveConversation(conv)

	res := &ApplyResult{}
	now := models.NewDateTime()
	for _, id := range draftIds {
		d, err := store.GetDraft(id)
		if err != nil || d == nil || d.ConversationId != conv.Id || d.CreateId != userId || d.Status != DraftPending {
			res.Failed = append(res.Failed, map[string]string{"draftId": id, "reason": ReasonNotFound})
			continue
		}
		if !time.Time(d.ExpireTime).IsZero() && time.Time(d.ExpireTime).Before(time.Now()) {
			res.Failed = append(res.Failed, map[string]string{"draftId": id, "reason": ReasonDraftExpired})
			continue
		}
		pid, err := applyMutatingFn(userId, d.ToolName, d.Payload)
		if err != nil {
			_ = store.SaveAudit(&models.AgentAudit{
				Id: newHexID(), ConversationId: conv.Id, DraftId: d.Id, ToolName: d.ToolName,
				ProductId: pid, Action: "apply", Success: false, Error: err.Error(),
				CreateId: userId, CreateTime: now,
			})
			// keep pending
			res.Failed = append(res.Failed, map[string]string{"draftId": d.Id, "reason": err.Error()})
			continue
		}
		d.Status = DraftApplied
		d.ProductId = pid
		_ = store.SaveDraft(d)
		ev := map[string]any{"draftId": d.Id, "tool": d.ToolName, "productId": pid, "appliedTime": now}
		raw, _ := json.Marshal(ev)
		_ = store.SaveMessage(&models.AgentMessage{
			Id: newHexID(), ConversationId: conv.Id, Role: RoleEvent, EventType: EventApplied,
			Payload: string(raw), Content: "已应用 " + d.ToolName, ToolName: d.ToolName,
			DraftId: d.Id, ProductId: pid, CreateId: userId, CreateTime: now,
		})
		_ = store.SaveMessage(&models.AgentMessage{
			Id: newHexID(), ConversationId: conv.Id, Role: RoleTool,
			Payload: toolPayload(d.ToolCallId, d.ToolName, string(raw)), Content: string(raw), ToolName: d.ToolName,
			ToolCallId: d.ToolCallId, DraftId: d.Id, ProductId: pid, CreateId: userId, CreateTime: now,
		})
		_ = store.SaveAudit(&models.AgentAudit{
			Id: newHexID(), ConversationId: conv.Id, DraftId: d.Id, ToolName: d.ToolName,
			ProductId: pid, Action: "apply", Success: true, CreateId: userId, CreateTime: now,
		})
		res.Applied = append(res.Applied, map[string]string{"draftId": d.Id, "tool": d.ToolName, "productId": pid})
	}

	all, _ := store.ListDrafts(conv.Id)
	pending := PendingDrafts(all)
	for _, d := range pending {
		res.PendingDrafts = append(res.PendingDrafts, d.Id)
	}
	if len(pending) > 0 {
		conv.RunStatus = RunAwaitingConfirm
		conv.NeedsResume = false
	} else {
		conv.RunStatus = RunIdle
		conv.NeedsResume = resume && len(res.Failed) == 0 && len(res.Applied) > 0
	}
	_ = store.SaveConversation(conv)
	res.RunStatus = conv.RunStatus
	res.NeedsResume = conv.NeedsResume
	return res, nil
}

func RejectDrafts(store Store, userId int64, conv *models.AgentConversation, draftIds []string) (*ApplyResult, error) {
	if conv.RunStatus == RunRunning || conv.RunStatus == RunApplying {
		return nil, fmt.Errorf("%s", ReasonRunActive)
	}
	if conv.RunStatus != RunAwaitingConfirm {
		return nil, fmt.Errorf("%s", ReasonNotAwaiting)
	}
	now := models.NewDateTime()
	res := &ApplyResult{}
	for _, id := range draftIds {
		d, err := store.GetDraft(id)
		if err != nil || d == nil || d.ConversationId != conv.Id || d.CreateId != userId || d.Status != DraftPending {
			res.Failed = append(res.Failed, map[string]string{"draftId": id, "reason": ReasonNotFound})
			continue
		}
		d.Status = DraftRejected
		_ = store.SaveDraft(d)
		ev := map[string]any{"draftId": d.Id, "tool": d.ToolName, "rejected": true}
		raw, _ := json.Marshal(ev)
		_ = store.SaveMessage(&models.AgentMessage{
			Id: newHexID(), ConversationId: conv.Id, Role: RoleEvent, EventType: EventRejected,
			Payload: string(raw), Content: "已拒绝 " + d.ToolName, ToolName: d.ToolName,
			DraftId: d.Id, CreateId: userId, CreateTime: now,
		})
		_ = store.SaveMessage(&models.AgentMessage{
			Id: newHexID(), ConversationId: conv.Id, Role: RoleTool,
			Payload: toolPayload(d.ToolCallId, d.ToolName, `{"applied":false,"reason":"rejected"}`), Content: "rejected", ToolName: d.ToolName,
			ToolCallId: d.ToolCallId, DraftId: d.Id, CreateId: userId, CreateTime: now,
		})
		res.Applied = append(res.Applied, map[string]string{"draftId": d.Id, "tool": d.ToolName})
	}
	all, _ := store.ListDrafts(conv.Id)
	pending := PendingDrafts(all)
	for _, d := range pending {
		res.PendingDrafts = append(res.PendingDrafts, d.Id)
	}
	conv.RunStatus = RunStatusForDrafts(len(pending))
	conv.NeedsResume = false
	_ = store.SaveConversation(conv)
	res.RunStatus = conv.RunStatus
	return res, nil
}

func QueueConfirmDraft(store Store, userId int64, conv *models.AgentConversation, toolCallId, toolName, payload, preview string, productId string) (*models.AgentDraft, error) {
	now := models.NewDateTime()
	d := &models.AgentDraft{
		Id: newHexID(), ConversationId: conv.Id, ToolCallId: toolCallId, ToolName: toolName,
		ProductId: productId, Payload: payload, Preview: preview, Status: DraftPending,
		ExpireTime: models.DateTime(time.Now().Add(24 * time.Hour)),
		CreateId: userId, CreateTime: now,
	}
	if err := store.SaveDraft(d); err != nil {
		return nil, err
	}
	ev := map[string]any{"draftId": d.Id, "tool": toolName, "preview": json.RawMessage(preview)}
	raw, _ := json.Marshal(ev)
	_ = store.SaveMessage(&models.AgentMessage{
		Id: newHexID(), ConversationId: conv.Id, Role: RoleEvent, EventType: EventConfirmRequired,
		Payload: string(raw), Content: "待确认 " + toolName, ToolName: toolName,
		DraftId: d.Id, ProductId: productId, CreateId: userId, CreateTime: now,
	})
	conv.RunStatus = RunAwaitingConfirm
	conv.NeedsResume = false
	_ = store.SaveConversation(conv)
	return d, nil
}
