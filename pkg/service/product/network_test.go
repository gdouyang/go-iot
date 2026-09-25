package product

import (
	"errors"
	"testing"

	"go-iot/pkg/models"
	"go-iot/pkg/network"

	"github.com/stretchr/testify/require"
)

func TestAssertStartableRejectsClientTypes(t *testing.T) {
	err := AssertStartable(&models.ProductModel{Product: models.Product{
		Id: "P1", NetworkType: string(network.MODBUS), Script: "function OnMessage(c){}",
	}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "客户端")

	err = AssertStartable(&models.ProductModel{Product: models.Product{
		Id: "P1", NetworkType: string(network.MQTT_BROKER),
	}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "编解码")

	err = AssertStartable(&models.ProductModel{Product: models.Product{
		Id: "P1", NetworkType: string(network.MQTT_BROKER), Script: "function OnMessage(c){}",
	}})
	require.NoError(t, err)
}

func TestStartNetworkUsesHooks(t *testing.T) {
	origGet := getProductMust
	origNet := getNetworkByProduct
	origBind := bindNetworkProduct
	origConv := convertCodecNetwork
	origStart := startServer
	origServer := getServer
	origUpdate := updateNetworkRow
	t.Cleanup(func() {
		getProductMust = origGet
		getNetworkByProduct = origNet
		bindNetworkProduct = origBind
		convertCodecNetwork = origConv
		startServer = origStart
		getServer = origServer
		updateNetworkRow = origUpdate
	})

	getProductMust = func(id string) (*models.ProductModel, error) {
		return &models.ProductModel{Product: models.Product{
			Id: id, CreateId: 7, NetworkType: string(network.TCP_SERVER),
			Script: "function OnMessage(c){}",
		}}, nil
	}
	getNetworkByProduct = func(productId string) (*models.Network, error) {
		return &models.Network{Id: 3, ProductId: productId, Type: string(network.TCP_SERVER), Port: 9010}, nil
	}
	getServer = func(string) network.NetServer { return nil }
	convertCodecNetwork = func(nw models.Network) (network.NetworkConf, error) {
		return network.NetworkConf{ProductId: nw.ProductId, Type: nw.Type, Port: nw.Port}, nil
	}
	var started network.NetworkConf
	startServer = func(conf network.NetworkConf) error {
		started = conf
		return nil
	}
	var savedState string
	updateNetworkRow = func(ob *models.Network) error {
		savedState = ob.State
		return nil
	}

	require.NoError(t, StartNetwork(7, "P1"))
	require.Equal(t, "P1", started.ProductId)
	require.Equal(t, models.Runing, savedState)

	getProductMust = func(id string) (*models.ProductModel, error) {
		return &models.ProductModel{Product: models.Product{
			Id: id, CreateId: 7, NetworkType: string(network.MODBUS), Script: "s",
		}}, nil
	}
	err := StartNetwork(7, "P1")
	require.Error(t, err)
}

func TestUpdateNetworkConfigRejectsBadJSON(t *testing.T) {
	origGet := getProductMust
	t.Cleanup(func() { getProductMust = origGet })
	getProductMust = func(id string) (*models.ProductModel, error) {
		return &models.ProductModel{Product: models.Product{Id: id, CreateId: 1, NetworkType: "MQTT_CLIENT"}}, nil
	}
	_, err := UpdateNetworkConfig(1, "P1", "", "{not-json", 0)
	require.Error(t, err)
}

func TestStartNetworkOwnerCheck(t *testing.T) {
	origGet := getProductMust
	t.Cleanup(func() { getProductMust = origGet })
	getProductMust = func(id string) (*models.ProductModel, error) {
		return &models.ProductModel{Product: models.Product{Id: id, CreateId: 1, NetworkType: "TCP_SERVER", Script: "s"}}, nil
	}
	err := StartNetwork(2, "P1")
	require.ErrorIs(t, err, ErrNotOwner)
}

func TestStopNetworkWhenNotRunning(t *testing.T) {
	origGet := getProductMust
	origServer := getServer
	t.Cleanup(func() {
		getProductMust = origGet
		getServer = origServer
	})
	getProductMust = func(id string) (*models.ProductModel, error) {
		return &models.ProductModel{Product: models.Product{Id: id, CreateId: 1}}, nil
	}
	getServer = func(string) network.NetServer { return nil }
	err := StopNetwork(1, "P1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not runing")
}

func TestAssertDeployableEmptyTSL(t *testing.T) {
	_, err := AssertDeployable(&models.ProductModel{Product: models.Product{Id: "P1"}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "物模型")
}

func TestStartServerHookError(t *testing.T) {
	origGet := getProductMust
	origNet := getNetworkByProduct
	origConv := convertCodecNetwork
	origStart := startServer
	origServer := getServer
	t.Cleanup(func() {
		getProductMust = origGet
		getNetworkByProduct = origNet
		convertCodecNetwork = origConv
		startServer = origStart
		getServer = origServer
	})
	getProductMust = func(id string) (*models.ProductModel, error) {
		return &models.ProductModel{Product: models.Product{
			Id: id, CreateId: 1, NetworkType: "TCP_SERVER", Script: "function OnMessage(c){}",
		}}, nil
	}
	getNetworkByProduct = func(productId string) (*models.Network, error) {
		return &models.Network{Id: 1, ProductId: productId, Type: "TCP_SERVER", Port: 9011}, nil
	}
	getServer = func(string) network.NetServer { return nil }
	convertCodecNetwork = func(nw models.Network) (network.NetworkConf, error) {
		return network.NetworkConf{ProductId: nw.ProductId}, nil
	}
	startServer = func(network.NetworkConf) error { return errors.New("bind: address already in use") }
	err := StartNetwork(1, "P1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "address already in use")
}
