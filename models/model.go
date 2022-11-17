package models

import (
	"encoding/json"
	"go-iot/ruleengine"

	"github.com/beego/beego/v2/core/logs"
)

// device
type DeviceModel struct {
	Device
	Metaconfig map[string]string `json:"metaconfig,omitempty"`
}

func (d *DeviceModel) FromEnitty(en Device) {
	d.Device = en
	if len(en.Metaconfig) > 0 {
		m := map[string]string{}
		err := json.Unmarshal([]byte(en.Metaconfig), &m)
		if err != nil {
			logs.Error(err)
		}
		d.Metaconfig = m
	}
}

func (d *DeviceModel) ToEnitty() Device {
	en := d.Device
	v, err := json.Marshal(d.Metaconfig)
	if err != nil {
		logs.Error(err)
	} else {
		en.Metaconfig = string(v)
	}
	return en
}

// product
type ProductModel struct {
	Product
	Metaconfig []ProductMetaConfig `json:"metaconfig,omitempty"`
}

func (d *ProductModel) FromEnitty(en Product) {
	d.Product = en
	if len(en.Metaconfig) > 0 {
		m := []ProductMetaConfig{}
		err := json.Unmarshal([]byte(en.Metaconfig), &m)
		if err != nil {
			logs.Error(err)
		}
		d.Metaconfig = m
	}
}

func (d *ProductModel) ToEnitty() Product {
	en := d.Product
	v, err := json.Marshal(d.Metaconfig)
	if err != nil {
		logs.Error(err)
	} else {
		en.Metaconfig = string(v)
	}
	return en
}

type ProductMetaConfig struct {
	Property string `json:"property,omitempty"`
	Type     string `json:"type,omitempty"`
	Value    string `json:"value,omitempty"`
	Desc     string `json:"desc,omitempty"`
}

// scene
type SceneModel struct {
	Scene
	DeviceIds []string                `json:"deviceIds"`
	Trigger   ruleengine.SceneTrigger `json:"trigger"`
	Actions   []ruleengine.Action     `json:"actions"`
}

func (d *SceneModel) FromEnitty(en Scene) {
	d.Scene = en
	if len(en.Trigger) > 0 {
		m := ruleengine.SceneTrigger{}
		err := json.Unmarshal([]byte(en.Trigger), &m)
		if err != nil {
			logs.Error(err)
		}
		d.Trigger = m
	}
	if len(en.Actions) > 0 {
		m := []ruleengine.Action{}
		err := json.Unmarshal([]byte(en.Actions), &m)
		if err != nil {
			logs.Error(err)
		}
		d.Actions = m
	}
}

func (d *SceneModel) ToEnitty() Scene {
	en := d.Scene
	// trigger
	v, err := json.Marshal(d.Trigger)
	if err != nil {
		logs.Error(err)
	} else {
		en.Trigger = string(v)
	}
	// actions
	v, err = json.Marshal(d.Actions)
	if err != nil {
		logs.Error(err)
	} else {
		en.Actions = string(v)
	}
	return en
}
