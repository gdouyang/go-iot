package models

import (
	"encoding/json"

	"github.com/beego/beego/v2/core/logs"
)

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
