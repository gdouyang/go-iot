package agent

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"go-iot/pkg/models"

	"github.com/stretchr/testify/require"
)

func TestBuildPreviewCreateHasNilOld(t *testing.T) {
	orig := lookupProduct
	t.Cleanup(func() { lookupProduct = orig })
	lookupProduct = func(id string) (*models.ProductModel, error) {
		return nil, errors.New("missing")
	}
	raw, pid := BuildPreview(writeCtx(1), ToolCreateProduct, json.RawMessage(`{"id":"A","name":"n","networkType":"MQTT_BROKER"}`))
	require.Equal(t, "A", pid)
	var doc previewDoc
	require.NoError(t, json.Unmarshal([]byte(raw), &doc))
	require.Nil(t, doc.Old)
	require.NotNil(t, doc.New)
}

func TestBuildPreviewSaveTSLIncludesOldAndPublished(t *testing.T) {
	orig := lookupProduct
	t.Cleanup(func() { lookupProduct = orig })
	lookupProduct = func(id string) (*models.ProductModel, error) {
		return &models.ProductModel{Product: models.Product{
			Id: id, CreateId: 7, State: true,
			Metadata: `{"properties":[{"id":"t","name":"t","type":"int"}]}`,
		}}, nil
	}
	raw, pid := BuildPreview(writeCtx(7), ToolSaveTSL, json.RawMessage(`{"productId":"P1","tsl":{"properties":[]}}`))
	require.Equal(t, "P1", pid)
	var doc previewDoc
	require.NoError(t, json.Unmarshal([]byte(raw), &doc))
	require.True(t, doc.Published)
	require.NotEmpty(t, doc.Warning)
	oldMap, ok := doc.Old.(map[string]any)
	require.True(t, ok)
	require.NotNil(t, oldMap["tsl"])
}

func TestPendingDraftsSkipsExpired(t *testing.T) {
	expired := models.DateTime(time.Now().Add(-time.Hour))
	list := []models.AgentDraft{
		{Id: "a", Status: DraftPending, ExpireTime: expired},
		{Id: "b", Status: DraftPending},
	}
	got := PendingDrafts(list)
	require.Len(t, got, 1)
	require.Equal(t, "b", got[0].Id)
}
