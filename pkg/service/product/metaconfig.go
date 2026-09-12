package product

import (
	"errors"
	"fmt"
	"strings"

	"go-iot/pkg/core"
	"go-iot/pkg/models"
	"go-iot/pkg/network"
)

var getMetaSchema = network.GetNetworkMetaConfig

func MetaSchema(networkType string) []core.MetaConfig {
	return append([]core.MetaConfig{}, getMetaSchema(networkType).MetaConfigs...)
}

func GetMetaconfig(userId int64, productId string) ([]core.MetaConfig, error) {
	exist, err := ownedProduct(userId, productId)
	if err != nil {
		return nil, err
	}
	return DisplayMetaconfig(exist.Metaconfig, exist.NetworkType), nil
}

func DisplayMetaconfig(exist []core.MetaConfig, networkType string) []core.MetaConfig {
	merged, _ := MergeMeta(exist, networkType, nil)
	return merged
}

// MergeMeta overlays set onto schema+existing rows. set nil/empty keeps keys and fills missing builtins.
func MergeMeta(exist []core.MetaConfig, networkType string, set map[string]string) ([]core.MetaConfig, error) {
	byProp := map[string]core.MetaConfig{}
	var order []string
	add := func(it core.MetaConfig) {
		p := strings.TrimSpace(it.Property)
		if p == "" {
			return
		}
		it.Property = p
		if _, ok := byProp[p]; !ok {
			order = append(order, p)
		}
		byProp[p] = it
	}
	for _, it := range MetaSchema(networkType) {
		add(it)
	}
	for _, it := range exist {
		add(it)
	}
	for k, v := range set {
		k = strings.TrimSpace(k)
		if k == "" {
			return nil, errors.New("config key is empty")
		}
		cur, ok := byProp[k]
		if !ok {
			cur = core.MetaConfig{Property: k, Type: "string"}
			order = append(order, k)
		}
		cur.Value = v
		byProp[k] = cur
	}
	out := make([]core.MetaConfig, 0, len(order))
	for _, k := range order {
		out = append(out, byProp[k])
	}
	return out, nil
}

func SaveMetaconfig(userId int64, productId string, set map[string]string) ([]core.MetaConfig, error) {
	if len(set) == 0 {
		return nil, errors.New("config set is empty")
	}
	exist, err := ownedProduct(userId, productId)
	if err != nil {
		return nil, err
	}
	merged, err := MergeMeta(exist.Metaconfig, exist.NetworkType, set)
	if err != nil {
		return nil, err
	}
	patch := &models.ProductModel{}
	patch.Id = productId
	patch.Metaconfig = merged
	if err := updateProduct(patch); err != nil {
		return nil, fmt.Errorf("save metaconfig: %w", err)
	}
	return merged, nil
}
