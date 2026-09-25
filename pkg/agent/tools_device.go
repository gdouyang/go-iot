package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"go-iot/pkg/agent/client"
	"go-iot/pkg/core"
	"go-iot/pkg/models"
	devicemd "go-iot/pkg/models/device"
	productsvc "go-iot/pkg/service/product"
)

var lookupDevice = func(id string) (*models.DeviceModel, error) {
	return devicemd.GetDevice(id)
}

var addDevice = func(ob *models.DeviceModel) error {
	return devicemd.AddDevice(ob)
}

type createDeviceTool struct{}

func (createDeviceTool) Name() string   { return ToolCreateDevice }
func (createDeviceTool) Mutating() bool { return true }
func (createDeviceTool) Spec() client.CompatTool {
	return specOf(ToolCreateDevice, "在指定产品下创建新设备实例（适用于 MQTT、TCP、HTTP、WebSocket、CoAP、MODBUS 等各类网络接入协议）。属于写操作，执行前需确认。",
		`{"type":"object","properties":{"id":{"type":"string","description":"设备全局唯一ID（支持字母、数字、_、-，最长32位），作为设备在网络接入时的身份标识（如 MQTT clientId、TCP 注册包设备号、Modbus 采集实例等）"},"name":{"type":"string","description":"设备显示名称"},"productId":{"type":"string","description":"所属产品ID"},"desc":{"type":"string","description":"设备描述说明，选填"}},"required":["id","name","productId"]}`)
}
func (createDeviceTool) Validate(ctx ToolContext, args json.RawMessage) error {
	p, err := parseCreateDevice(args)
	if err != nil {
		return err
	}
	if err := requireOwnedProduct(ctx.UserId, p.ProductId); err != nil {
		return err
	}
	exist, err := lookupDevice(p.Id)
	if err == nil && exist != nil {
		return fmt.Errorf("device already exists")
	}
	return nil
}
func (createDeviceTool) Execute(ctx ToolContext, args json.RawMessage) (any, string, error) {
	p, err := parseCreateDevice(args)
	if err != nil {
		return nil, "", err
	}
	ob, err := lookupProduct(p.ProductId)
	if err != nil {
		return nil, p.ProductId, err
	}
	if err := productsvc.AssertOwner(ob, ctx.UserId); err != nil {
		return nil, p.ProductId, err
	}
	dev := &models.DeviceModel{Device: models.Device{
		Id: p.Id, Name: p.Name, ProductId: p.ProductId, Desc: p.Desc,
		DeviceType: core.DEVICE, CreateId: ctx.UserId,
	}}
	if err := addDevice(dev); err != nil {
		return nil, p.ProductId, err
	}
	return map[string]any{"ok": true, "deviceId": devicemd.NormalizeId(p.Id), "productId": p.ProductId}, p.ProductId, nil
}

type createDeviceArgs struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	ProductId string `json:"productId"`
	Desc      string `json:"desc"`
}

func parseCreateDevice(args json.RawMessage) (createDeviceArgs, error) {
	var p createDeviceArgs
	if len(bytesTrimSpace(args)) == 0 || !json.Valid(args) {
		return p, fmt.Errorf("invalid tool arguments")
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return p, err
	}
	p.Id = devicemd.NormalizeId(p.Id)
	p.Name = strings.TrimSpace(p.Name)
	p.ProductId = strings.TrimSpace(p.ProductId)
	if p.Id == "" || p.Name == "" || p.ProductId == "" {
		return p, fmt.Errorf("id, name, productId are required")
	}
	if !devicemd.DeviceIdValid(p.Id) {
		return p, fmt.Errorf("设备ID格式错误")
	}
	return p, nil
}

var listDevices = func(productId string, pageNum, pageSize int) (*models.PageResult[models.Device], error) {
	if pageNum <= 0 {
		pageNum = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	pq := &models.PageQuery{
		PageNum:  pageNum,
		PageSize: pageSize,
		Condition: []core.SearchTerm{
			{Key: "productId", Value: productId, Oper: "EQ"},
		},
	}
	return devicemd.PageDevice(pq, nil)
}

type getDeviceTool struct{}

func (getDeviceTool) Name() string   { return ToolGetDevice }
func (getDeviceTool) Mutating() bool { return false }
func (getDeviceTool) Spec() client.CompatTool {
	return specOf(ToolGetDevice, "根据设备ID查询单个设备的详细信息及实时在线/离线状态。已连接设备会附带连接详情。",
		`{"type":"object","properties":{"deviceId":{"type":"string","description":"设备唯一ID"}},"required":["deviceId"]}`)
}

func (getDeviceTool) Validate(ctx ToolContext, args json.RawMessage) error {
	var p struct {
		DeviceId string `json:"deviceId"`
	}
	if len(bytesTrimSpace(args)) == 0 || !json.Valid(args) {
		return fmt.Errorf("invalid tool arguments")
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.DeviceId) == "" {
		return fmt.Errorf("deviceId is required")
	}
	return nil
}

func (getDeviceTool) Execute(ctx ToolContext, args json.RawMessage) (any, string, error) {
	var p struct {
		DeviceId string `json:"deviceId"`
	}
	_ = json.Unmarshal(args, &p)
	deviceId := strings.TrimSpace(p.DeviceId)

	dev, err := lookupDevice(deviceId)
	if err != nil || dev == nil {
		return map[string]any{"ok": false, "message": "设备不存在: " + deviceId}, "", nil
	}

	ob, err := lookupProduct(dev.ProductId)
	if err == nil && ob != nil {
		if err := productsvc.AssertOwner(ob, ctx.UserId); err != nil {
			return nil, dev.ProductId, err
		}
	}

	state := dev.State
	if state != core.NoActive {
		state = core.GetDeviceState(dev.Id, dev.ProductId)
	}
	online := state == core.ONLINE

	var conInfo map[string]any
	if sess := core.GetSession(dev.Id); sess != nil {
		conInfo = sess.GetConInfo()
	}

	res := map[string]any{
		"ok":         true,
		"deviceId":   dev.Id,
		"name":       dev.Name,
		"productId":  dev.ProductId,
		"state":      state,
		"online":     online,
		"deviceType": dev.DeviceType,
		"desc":       dev.Desc,
		"createTime": dev.CreateTime,
	}
	if conInfo != nil {
		res["connection"] = conInfo
	}
	return res, dev.ProductId, nil
}

type listDevicesTool struct{}

func (listDevicesTool) Name() string   { return ToolListDevices }
func (listDevicesTool) Mutating() bool { return false }
func (listDevicesTool) Spec() client.CompatTool {
	return specOf(ToolListDevices, "分页查询指定产品下的设备列表，包含每个设备的实时在线/离线状态。",
		`{"type":"object","properties":{"productId":{"type":"string","description":"所属产品ID"},"pageNum":{"type":"integer","description":"页码，默认 1"},"pageSize":{"type":"integer","description":"每页数量，默认 20，最大 100"}},"required":["productId"]}`)
}

func (listDevicesTool) Validate(ctx ToolContext, args json.RawMessage) error {
	var p struct {
		ProductId string `json:"productId"`
	}
	if len(bytesTrimSpace(args)) == 0 || !json.Valid(args) {
		return fmt.Errorf("invalid tool arguments")
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return err
	}
	if strings.TrimSpace(p.ProductId) == "" {
		return fmt.Errorf("productId is required")
	}
	return nil
}

func (listDevicesTool) Execute(ctx ToolContext, args json.RawMessage) (any, string, error) {
	var p struct {
		ProductId string `json:"productId"`
		PageNum   int    `json:"pageNum"`
		PageSize  int    `json:"pageSize"`
	}
	_ = json.Unmarshal(args, &p)
	productId := strings.TrimSpace(p.ProductId)

	if err := requireOwnedProduct(ctx.UserId, productId); err != nil {
		return nil, productId, err
	}

	pr, err := listDevices(productId, p.PageNum, p.PageSize)
	if err != nil {
		return nil, productId, err
	}

	var items []map[string]any
	if pr != nil && len(pr.List) > 0 {
		items = make([]map[string]any, 0, len(pr.List))
		for _, d := range pr.List {
			state := d.State
			if state != core.NoActive {
				state = core.GetDeviceState(d.Id, d.ProductId)
			}
			items = append(items, map[string]any{
				"deviceId":   d.Id,
				"name":       d.Name,
				"state":      state,
				"online":     state == core.ONLINE,
				"deviceType": d.DeviceType,
				"createTime": d.CreateTime,
			})
		}
	} else {
		items = []map[string]any{}
	}

	total := int64(0)
	pageNum := 1
	pageSize := 20
	if pr != nil {
		total = pr.TotalCount
		pageNum = pr.PageNum
		pageSize = pr.PageSize
	}

	return map[string]any{
		"ok":        true,
		"productId": productId,
		"total":     total,
		"pageNum":   pageNum,
		"pageSize":  pageSize,
		"list":      items,
	}, productId, nil
}
