package codec

type DeviceLifecycle interface {
	// 设备新增
	OnCreate(ctx Context) error
	// 设备删除
	OnDelete(ctx Context) error
	// 设备修改
	OnUpdate(ctx Context) error
	//
	OnStateChecker(ctx Context) error
}

var deviceManagerIns DeviceManager = DeviceManager{}
var productManager ProductManager = ProductManager{}

func GetDeviceManager() *DeviceManager {
	return &deviceManagerIns
}

func GetProductManager() *ProductManager {
	return &productManager
}

type DeviceManager struct {
	m map[string]Device
}

func (dm *DeviceManager) GetDevice(deviceId string) Device {
	device := dm.m[deviceId]
	return device
}

func (dm *DeviceManager) PutDevice(deviceId string, device Device) {
	dm.m[deviceId] = device
}

type ProductManager struct {
	m map[string]Product
}

func (pm *ProductManager) GetProduct(productId string) Product {
	product := pm.m[productId]
	return product
}

type defaultDevice struct {
	session Session
	data    map[string]interface{}
	config  map[string]interface{}
}

func (device *defaultDevice) GetSession() Session {
	return device.session
}
func (device *defaultDevice) GetData() map[string]interface{} {
	return device.data
}
func (device *defaultDevice) GetConfig() map[string]interface{} {
	return device.config
}

type DefaultProdeuct struct {
	config map[string]interface{}
}

func (product *DefaultProdeuct) GetConfig() map[string]interface{} {
	return product.config
}
