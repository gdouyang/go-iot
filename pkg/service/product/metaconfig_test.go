package product

import (
	"testing"

	"go-iot/pkg/core"
	"go-iot/pkg/models"

	"github.com/stretchr/testify/require"
)

func TestMergeMetaFillsMQTTBrokerUsernamePassword(t *testing.T) {
	orig := getMetaSchema
	t.Cleanup(func() { getMetaSchema = orig })
	getMetaSchema = func(string) core.CodecMetaConfig {
		return core.CodecMetaConfig{MetaConfigs: []core.MetaConfig{
			{Property: "username", Type: "string", Buildin: true, Desc: "MQTT 用户名"},
			{Property: "password", Type: "password", Buildin: true, Desc: "MQTT 密码"},
		}}
	}
	out, err := MergeMeta(nil, "MQTT_BROKER", map[string]string{"username": "u1", "password": "p1"})
	require.NoError(t, err)
	require.Len(t, out, 2)
	require.Equal(t, "u1", out[0].Value)
	require.Equal(t, "p1", out[1].Value)
	require.True(t, out[0].Buildin)
	require.Equal(t, "password", out[1].Type)
}

func TestSaveMetaconfigRejectsEmptySet(t *testing.T) {
	_, err := SaveMetaconfig(1, "P1", nil)
	require.Error(t, err)
}

func TestSaveMetaconfigUsesHooks(t *testing.T) {
	origGet := getProductMust
	origUpd := updateProduct
	origSchema := getMetaSchema
	t.Cleanup(func() {
		getProductMust = origGet
		updateProduct = origUpd
		getMetaSchema = origSchema
	})
	getMetaSchema = func(string) core.CodecMetaConfig {
		return core.CodecMetaConfig{MetaConfigs: []core.MetaConfig{
			{Property: "username", Type: "string", Buildin: true},
		}}
	}
	getProductMust = func(id string) (*models.ProductModel, error) {
		return &models.ProductModel{Product: models.Product{Id: id, CreateId: 3, NetworkType: "MQTT_BROKER"}}, nil
	}
	var saved []core.MetaConfig
	updateProduct = func(ob *models.ProductModel) error {
		saved = ob.Metaconfig
		return nil
	}
	out, err := SaveMetaconfig(3, "P1", map[string]string{"username": "broker"})
	require.NoError(t, err)
	require.Equal(t, "broker", out[0].Value)
	require.Equal(t, "broker", saved[0].Value)
}
