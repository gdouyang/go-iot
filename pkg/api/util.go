package api

import (
	"go-iot/pkg/bootstrap"
	"go-iot/pkg/models"
	"go-iot/pkg/network"
	"go-iot/pkg/ruleengine"
)

// 兼容原包内调用；实现见 pkg/bootstrap。
func convertCodecNetwork(nw models.Network) (network.NetworkConf, error) {
	return bootstrap.ConvertCodecNetwork(nw)
}

func connectClientDevice(deviceId string) error {
	return bootstrap.ConnectClientDevice(deviceId)
}

func ruleModelToRuleExecutor(m *models.RuleModel) ruleengine.RuleExecutor {
	return bootstrap.RuleModelToRuleExecutor(m)
}
