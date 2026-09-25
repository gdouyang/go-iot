package agent

import (
	"sync/atomic"
	"testing"
	"time"

	"go-iot/pkg/models"

	"github.com/stretchr/testify/require"
)

func TestSaveConversationDoesNotTouchTitle(t *testing.T) {
	store := NewMemoryStore()
	conv := &models.AgentConversation{
		Id: "c-title", Title: DefaultConvTitle, Status: "active",
		RunStatus: RunRunning, CreateId: 1,
	}
	require.NoError(t, store.SaveConversation(conv))
	require.NoError(t, store.UpdateTitle(conv.Id, "MQTT温湿度"))

	// run 收尾整份回写（不带标题），不得把标题盖回默认值。
	conv.RunStatus = RunIdle
	require.NoError(t, store.SaveConversation(conv))
	got, _ := store.GetConversation(conv.Id)
	require.Equal(t, "MQTT温湿度", got.Title)
	require.Equal(t, RunIdle, got.RunStatus)
}

func TestUpdateTitleKeepsRunStatus(t *testing.T) {
	store := NewMemoryStore()
	conv := &models.AgentConversation{
		Id: "c-rename", Title: DefaultConvTitle, Status: "active",
		RunStatus: RunRunning, CreateId: 1, LastPromptTokens: 10,
	}
	require.NoError(t, store.SaveConversation(conv))
	require.NoError(t, store.UpdateTitle(conv.Id, "移动OneNet路灯"))
	got, _ := store.GetConversation(conv.Id)
	require.Equal(t, "移动OneNet路灯", got.Title)
	require.Equal(t, RunRunning, got.RunStatus)
	require.Equal(t, 10, got.LastPromptTokens)
}

func TestQueueConfirmDraftKeepsGeneratedTitle(t *testing.T) {
	store := NewMemoryStore()
	conv := &models.AgentConversation{
		Id: "c-confirm", Title: DefaultConvTitle, Status: "active",
		RunStatus: RunRunning, CreateId: 1,
	}
	require.NoError(t, store.SaveConversation(conv))
	require.NoError(t, store.UpdateTitle(conv.Id, "移动OneNet路灯"))

	_, err := QueueConfirmDraft(store, 1, conv, "call_1", "create_product", "{}", "{}", "", 1)
	require.NoError(t, err)
	got, _ := store.GetConversation(conv.Id)
	require.Equal(t, "移动OneNet路灯", got.Title)
	require.Equal(t, RunAwaitingConfirm, got.RunStatus)
}

// saveCountingStore 记录整份回写/心跳次数；心跳只能改 updateTime 字段，不得走 SaveConversation。
type saveCountingStore struct {
	*MemoryStore
	saves   int32
	touches int32
}

func (s *saveCountingStore) SaveConversation(c *models.AgentConversation) error {
	atomic.AddInt32(&s.saves, 1)
	return s.MemoryStore.SaveConversation(c)
}

func (s *saveCountingStore) TouchConversation(id string) error {
	atomic.AddInt32(&s.touches, 1)
	return s.MemoryStore.TouchConversation(id)
}

func TestHeartbeatOnlyTouchesUpdateTime(t *testing.T) {
	store := &saveCountingStore{MemoryStore: NewMemoryStore()}
	conv := &models.AgentConversation{
		Id: "c-hb", Title: DefaultConvTitle, Status: "active",
		RunStatus: RunRunning, CreateId: 1, LastPromptTokens: 10,
	}
	require.NoError(t, store.MemoryStore.SaveConversation(conv))
	before, _ := store.GetConversation(conv.Id)

	stop := startHeartbeat(store, conv, 5*time.Millisecond)
	defer stop()

	// 等心跳真的跑过一轮，再断言没有整份回写。
	deadline := time.Now().Add(2 * time.Second)
	for atomic.LoadInt32(&store.touches) == 0 {
		require.True(t, time.Now().Before(deadline), "heartbeat never fired")
		time.Sleep(2 * time.Millisecond)
	}
	// 心跳不得整份回写（否则会盖掉 run 刚写入的状态/计数）。
	require.Equal(t, int32(0), atomic.LoadInt32(&store.saves))
	got, _ := store.GetConversation(conv.Id)
	require.Equal(t, RunRunning, got.RunStatus)
	require.Equal(t, 10, got.LastPromptTokens)
	require.Equal(t, DefaultConvTitle, got.Title)
	require.False(t, time.Time(got.UpdateTime).Before(time.Time(before.UpdateTime)))
}
