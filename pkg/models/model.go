package models

import (
	"encoding/json"
	"go-iot/pkg/core"
	"go-iot/pkg/ruleengine"

	logs "go-iot/pkg/logger"
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
			logs.Errorf(err.Error())
		}
		d.Metaconfig = m
	}
}

func (d *DeviceModel) ToEnitty() Device {
	en := d.Device
	v, err := json.Marshal(d.Metaconfig)
	if err != nil {
		logs.Errorf(err.Error())
	} else {
		en.Metaconfig = string(v)
	}
	return en
}

// product
type ProductModel struct {
	Product
	Metaconfig []core.MetaConfig `json:"metaconfig,omitempty"`
}

func (p *ProductModel) ToProeuctOper() (*core.Product, error) {
	config := map[string]string{}
	for _, v := range p.Metaconfig {
		config[v.Property] = v.Value
	}
	productOpr, err := core.NewProduct(p.Id, config, p.StorePolicy, p.Metadata)
	if productOpr != nil {
		productOpr.NetworkType = p.NetworkType
	}
	return productOpr, err
}

func (d *ProductModel) FromEnitty(en Product) {
	d.Product = en
	if len(en.Metaconfig) > 0 {
		m := []core.MetaConfig{}
		err := json.Unmarshal([]byte(en.Metaconfig), &m)
		if err != nil {
			logs.Errorf(err.Error())
		}
		d.Metaconfig = m
	}
}

func (d *ProductModel) ToEnitty() Product {
	en := d.Product
	v, err := json.Marshal(d.Metaconfig)
	if err != nil {
		logs.Errorf(err.Error())
	} else {
		en.Metaconfig = string(v)
	}
	return en
}

// scene
type RuleModel struct {
	Rule
	DeviceIds []string            `json:"deviceIds"`
	Trigger   ruleengine.Trigger  `json:"trigger"`
	Actions   []ruleengine.Action `json:"actions"`
}

func (d *RuleModel) FromEnitty(en Rule) {
	d.Rule = en
	if len(en.Trigger) > 0 {
		m := ruleengine.Trigger{}
		err := json.Unmarshal([]byte(en.Trigger), &m)
		if err != nil {
			logs.Errorf(err.Error())
		}
		d.Trigger = m
	}
	if len(en.Actions) > 0 {
		m := []ruleengine.Action{}
		err := json.Unmarshal([]byte(en.Actions), &m)
		if err != nil {
			logs.Errorf(err.Error())
		}
		d.Actions = m
	}
}

func (d *RuleModel) ToEnitty() Rule {
	en := d.Rule
	// trigger
	v, err := json.Marshal(d.Trigger)
	if err != nil {
		logs.Errorf(err.Error())
	} else {
		en.Trigger = string(v)
	}
	// actions
	v, err = json.Marshal(d.Actions)
	if err != nil {
		logs.Errorf(err.Error())
	} else {
		en.Actions = string(v)
	}
	return en
}
