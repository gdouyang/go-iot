// Package bootstrap holds shared helpers used by process startup and HTTP admin APIs
// (network config conversion, outbound client connect, rule model mapping).
package bootstrap

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

// ConvertCodecNetwork builds runtime NetworkConf from a stored network row + product codec.
func ConvertCodecNetwork(nw models.Network) (network.NetworkConf, error) {
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

// ConnectClientDevice starts an outbound net client for an activated device.
func ConnectClientDevice(deviceId string) error {
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
	devoper := core.GetDevice(deviceId)
	if devoper == nil {
		return errors.New("devoper is nil")
	}
	conf, err := ConvertCodecNetwork(*nw)
	if err != nil {
		return err
	}
	if err = clients.Connect(deviceId, conf); err != nil {
		return err
	}
	return device.UpdateOnlineStatus(deviceId, core.ONLINE)
}

// RuleModelToRuleExecutor maps persistence model to ruleengine executor config.
func RuleModelToRuleExecutor(m *models.RuleModel) ruleengine.RuleExecutor {
	return ruleengine.RuleExecutor{
		Name:        m.Name,
		Type:        m.Type,
		ProductId:   m.ProductId,
		TriggerType: ruleengine.TriggerType(m.TriggerType),
		Cron:        m.Cron,
		Trigger:     m.Trigger,
		Actions:     m.Actions,
		DeviceIds:   m.DeviceIds,
	}
}
