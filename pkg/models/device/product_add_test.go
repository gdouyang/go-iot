package models

import (
	"errors"
	"testing"

	"go-iot/pkg/models"
	"go-iot/pkg/network"

	"github.com/stretchr/testify/require"
)

func TestAddProductRejectsUnknownNetworkType(t *testing.T) {
	var inserted bool
	origGet := getProductById
	origInsert := insertProductEntity
	t.Cleanup(func() {
		getProductById = origGet
		insertProductEntity = origInsert
	})
	getProductById = func(id string) (*models.ProductModel, error) { return nil, nil }
	insertProductEntity = func(entity *models.Product) error {
		inserted = true
		return nil
	}

	err := AddProduct(&models.ProductModel{
		Product: models.Product{Id: "bad-nt", Name: "bad", NetworkType: "MQTT"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "不支持的网络类型")
	require.False(t, inserted)

	err = AddProduct(&models.ProductModel{
		Product: models.Product{Id: "empty-nt", Name: "empty"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "networkType")
	require.False(t, inserted)
}

func TestAddProductNoOrphanWhenPortPoolEmpty(t *testing.T) {
	var inserted bool
	origGet := getProductById
	origLookup := lookupUnusedNetwork
	origInsert := insertProductEntity
	origBind := bindNetworkProduct
	origDel := deleteProductRow
	t.Cleanup(func() {
		getProductById = origGet
		lookupUnusedNetwork = origLookup
		insertProductEntity = origInsert
		bindNetworkProduct = origBind
		deleteProductRow = origDel
	})
	getProductById = func(id string) (*models.ProductModel, error) { return nil, nil }

	lookupUnusedNetwork = func() (*models.Network, error) {
		return nil, errors.New("没有空闲的端口可以使用")
	}
	insertProductEntity = func(entity *models.Product) error {
		inserted = true
		return nil
	}

	err := AddProduct(&models.ProductModel{
		Product: models.Product{
			Id:          "pool-full",
			Name:        "full",
			NetworkType: string(network.MQTT_BROKER),
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "没有空闲的端口可以使用")
	require.False(t, inserted, "must not insert a product row when the default port pool is exhausted")
}

func TestAddProductCompensatesDeleteWhenBindFails(t *testing.T) {
	var inserted, deleted bool
	origGet := getProductById
	origLookup := lookupUnusedNetwork
	origInsert := insertProductEntity
	origBind := bindNetworkProduct
	origDel := deleteProductRow
	t.Cleanup(func() {
		getProductById = origGet
		lookupUnusedNetwork = origLookup
		insertProductEntity = origInsert
		bindNetworkProduct = origBind
		deleteProductRow = origDel
	})
	getProductById = func(id string) (*models.ProductModel, error) { return nil, nil }

	lookupUnusedNetwork = func() (*models.Network, error) {
		return &models.Network{Port: 9010}, nil
	}
	insertProductEntity = func(entity *models.Product) error {
		inserted = true
		return nil
	}
	bindNetworkProduct = func(productId, networkType string) (*models.Network, error) {
		return nil, errors.New("bind failed")
	}
	deleteProductRow = func(id string) error {
		deleted = true
		require.Equal(t, "BIND-FAIL", id)
		return nil
	}

	err := AddProduct(&models.ProductModel{
		Product: models.Product{
			Id:          "bind-fail",
			Name:        "fail",
			NetworkType: string(network.MQTT_BROKER),
		},
	})
	require.Error(t, err)
	require.True(t, inserted)
	require.True(t, deleted, "bind failure must delete the product row")
}
