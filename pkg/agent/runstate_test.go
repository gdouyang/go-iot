package agent

import (
	"testing"
	"time"

	"go-iot/pkg/models"

	"github.com/stretchr/testify/require"
)

func TestRecoverStuckClearsStaleRunning(t *testing.T) {
	store := NewMemoryStore()
	c := &models.AgentConversation{
		Id: NewHexID(), Title: "t", Status: "active",
		RunStatus: RunRunning, CreateId: 1,
		UpdateTime: models.DateTime(time.Now().Add(-2 * time.Minute)),
	}
	require.NoError(t, store.SaveConversation(c))
	c.UpdateTime = models.DateTime(time.Now().Add(-2 * time.Minute))
	require.NoError(t, store.SaveConversation(c))
	// SaveConversation overwrites UpdateTime; set after save via copy
	got, _ := store.GetConversation(c.Id)
	got.RunStatus = RunRunning
	got.UpdateTime = models.DateTime(time.Now().Add(-2 * time.Minute))
	store.mu.Lock()
	store.convs[got.Id] = got
	store.mu.Unlock()

	fresh, _ := store.GetConversation(c.Id)
	RecoverStuck(store, fresh)
	require.Equal(t, RunIdle, fresh.RunStatus)
}

func TestRecoverStuckSkipsLocked(t *testing.T) {
	store := NewMemoryStore()
	c := &models.AgentConversation{
		Id: NewHexID(), RunStatus: RunRunning,
		UpdateTime: models.DateTime(time.Now().Add(-2 * time.Minute)),
	}
	require.True(t, TryLock(c.Id))
	t.Cleanup(func() { Unlock(c.Id) })
	RecoverStuck(store, c)
	require.Equal(t, RunRunning, c.RunStatus)
}

func TestConvRedisKey(t *testing.T) {
	require.Equal(t, "goiot:agent:conv:c123:lock", ConvRedisKey("c123", "lock"))
	require.Equal(t, "goiot:agent:conv:c123:seq", ConvRedisKey("c123", "seq"))
	require.Equal(t, "goiot:agent:conv:c123", ConvRedisKey("c123", ""))
	require.Equal(t, "goiot:agent:conv:cancel", CancelChannel)
}

