package api

import (
	"errors"
	"fmt"
	"go-iot/pkg/core"
	"go-iot/pkg/models"
	device "go-iot/pkg/models/device"
	networkmd "go-iot/pkg/models/network"
	"go-iot/pkg/network"
	"go-iot/pkg/network/clients"
	"go-iot/pkg/ruleengine"
)

func convertCodecNetwork(nw models.Network) (network.NetworkConf, error) {
	pro, err := device.GetProductMust(nw.ProductId)
	if err != nil {
		return network.NetworkConf{}, err
	}
	config := network.NetworkConf{
		Name:          nw.Name,
		Port:          nw.Port,
		ProductId:     nw.ProductId,
		Configuration: nw.Configuration,
		Script:        pro.Script,
		Type:          nw.Type,
		CodecId:       pro.CodecId,
		CertBase64:    nw.CertBase64,
		KeyBase64:     nw.KeyBase64,
	}
	return config, nil
}

func connectClientDevice(deviceId string) error {
	dev, err := device.GetDeviceMust(deviceId)
	if err != nil {
		return err
	}
	nw, err := networkmd.GetByProductId(dev.ProductId)
	if err != nil {
		return err
	}
	if nw == nil {
		return fmt.Errorf("product [%s] not have network config", dev.ProductId)
	}
	// 进行连接
	devoper := core.GetDevice(deviceId)
	if devoper == nil {
		return errors.New("devoper is nil")
	}
	conf, err := convertCodecNetwork(*nw)
	if err != nil {
		return err
	}
	err = clients.Connect(deviceId, conf)
	if err != nil {
		return err
	}
	err = device.UpdateOnlineStatus(deviceId, core.ONLINE)
	if err != nil {
		return err
	}
	return nil
}

func ruleModelToRuleExecutor(m *models.RuleModel) ruleengine.RuleExecutor {
	rule := ruleengine.RuleExecutor{
		Name:        m.Name,
		Type:        m.Type,
		ProductId:   m.ProductId,
		TriggerType: ruleengine.TriggerType(m.TriggerType),
		Cron:        m.Cron,
		Trigger:     m.Trigger,
		Actions:     m.Actions,
		DeviceIds:   m.DeviceIds,
	}
	return rule
}
