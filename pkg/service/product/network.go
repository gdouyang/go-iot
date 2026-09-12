package product

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"go-iot/pkg/models"
	"go-iot/pkg/network"
)

// GetNetwork returns the product's bound network row, binding a free port if needed.
// State is filled from the local server process (cluster peers are not queried).
func GetNetwork(userId int64, productId string) (*models.Network, error) {
	exist, err := ownedProduct(userId, productId)
	if err != nil {
		return nil, err
	}
	nw, err := getNetworkByProduct(productId)
	if err != nil {
		return nil, err
	}
	if nw == nil {
		nw, err = bindNetworkProduct(productId, exist.NetworkType)
		if err != nil {
			return nil, err
		}
	}
	if nw == nil {
		return nil, errors.New("product has no network config")
	}
	if getServer(productId) == nil {
		nw.State = models.Stop
	} else {
		nw.State = models.Runing
	}
	return nw, nil
}

// UpdateNetworkConfig updates the existing bound row. It does not allocate a new port
// from the pool; Port>0 updates the current row's listen port.
func UpdateNetworkConfig(userId int64, productId, name, configuration string, port int32) (*models.Network, error) {
	exist, err := ownedProduct(userId, productId)
	if err != nil {
		return nil, err
	}
	if configuration != "" && !json.Valid([]byte(configuration)) {
		return nil, errors.New("configuration must be JSON")
	}
	nw, err := GetNetwork(userId, productId)
	if err != nil {
		return nil, err
	}
	patch := &models.Network{
		Id:            nw.Id,
		ProductId:     productId,
		Type:          exist.NetworkType,
		Name:          name,
		Configuration: configuration,
		Port:          port,
	}
	if err := updateNetworkRow(patch); err != nil {
		return nil, err
	}
	return GetNetwork(userId, productId)
}

func AssertStartable(exist *models.ProductModel) error {
	if exist == nil {
		return errors.New("product not exist")
	}
	if network.IsNetClientType(exist.NetworkType) {
		return errors.New("客户端类型产品不能启动网络服务")
	}
	if len(strings.TrimSpace(exist.Script)) == 0 {
		return errors.New("产品没有配置编解码，请先配置")
	}
	return nil
}

// StartNetwork starts the northbound server in this process and marks the row running.
// Cluster peers are not started (same limitation as a local-only apply).
func StartNetwork(userId int64, productId string) error {
	exist, err := ownedProduct(userId, productId)
	if err != nil {
		return err
	}
	if err := AssertStartable(exist); err != nil {
		return err
	}
	nw, err := GetNetwork(userId, productId)
	if err != nil {
		return err
	}
	if len(nw.Type) == 0 {
		return errors.New("产品没有配置网络类型，请先配置")
	}
	if getServer(productId) != nil {
		return errors.New("network is runing")
	}
	conf, err := convertCodecNetwork(*nw)
	if err != nil {
		return err
	}
	if err := startServer(conf); err != nil {
		return err
	}
	nw.State = models.Runing
	return updateNetworkRow(nw)
}

func StopNetwork(userId int64, productId string) error {
	if _, err := ownedProduct(userId, productId); err != nil {
		return err
	}
	if getServer(productId) == nil {
		return errors.New("network is not runing")
	}
	if err := stopServer(productId); err != nil {
		return err
	}
	nw, err := getNetworkByProduct(productId)
	if err != nil {
		return err
	}
	if nw == nil {
		return nil
	}
	nw.State = models.Stop
	return updateNetworkRow(nw)
}

func ownedProduct(userId int64, productId string) (*models.ProductModel, error) {
	id := strings.TrimSpace(productId)
	if id == "" {
		return nil, fmt.Errorf("productId is required")
	}
	exist, err := getProductMust(id)
	if err != nil {
		return nil, err
	}
	if err := AssertOwner(exist, userId); err != nil {
		return nil, err
	}
	return exist, nil
}
