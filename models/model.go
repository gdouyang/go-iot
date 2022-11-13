package models

import (
	"encoding/json"

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
	Triggers []SceneTrigger `json:"triggers"`
	Actions  []Action       `json:"actions"`
}

func (d *SceneModel) FromEnitty(en Scene) {
	d.Scene = en
	if len(en.Triggers) > 0 {
		m := []SceneTrigger{}
		err := json.Unmarshal([]byte(en.Triggers), &m)
		if err != nil {
			logs.Error(err)
		}
		d.Triggers = m
	}
	if len(en.Actions) > 0 {
		m := []Action{}
		err := json.Unmarshal([]byte(en.Actions), &m)
		if err != nil {
			logs.Error(err)
		}
		d.Actions = m
	}
}

func (d *SceneModel) ToEnitty() Scene {
	en := d.Scene
	// triggers
	v, err := json.Marshal(d.Triggers)
	if err != nil {
		logs.Error(err)
	} else {
		en.Triggers = string(v)
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

type TriggerType string

const (
	TriggerTypeDevice TriggerType = "device"
	TriggerTypeTimer  TriggerType = "timer"
)

type SceneTrigger struct {
	Type   TriggerType        `json:"type"`
	Device SceneTriggerDevice `json:"device,omitempty"`
	Cron   string             `json:"cron,omitempty"`
}

type SceneTriggerDevice struct {
	ShakeLimit ShakeLimit        `json:"shakeLimit"` // 防抖限制
	Type       string            `json:"type"`       // 触发消息类型
	ModelId    string            `json:"modelId"`    // 物模型表示,如:属性ID,事件ID
	Filters    []ConditionFilter `json:"filters"`    // 条件
	ProductId  string            `json:"productId"`
	DeviceId   string            `json:"deviceId"`
}

type ConditionFilter struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Operator string `json:"operator"`
}

// 抖动限制
type ShakeLimit struct {
	Enabled    bool  `json:"enabled"`
	Time       int32 `json:"time"`
	Threshold  int32 `json:"threshold"`
	AlarmFirst bool  `json:"alarmFirst"`
}

// 执行
type Action struct {
	Executor      string                 `json:"executor"`      // 执行器
	Configuration map[string]interface{} `json:"configuration"` // 执行器配置
}
