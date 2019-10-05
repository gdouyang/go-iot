package models

import (
	"encoding/json"

	"github.com/astaxie/beego"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// 设备
type Device struct {
	Id   string `json:"id"`
	Sn   string `json:"sn"`
	Name string `json:"name"`
}

// 分页查询设备
func ListDevice(ob *PageQuery) *PageResult {
	var pr *PageResult
	var dev Device
	json.Unmarshal(ob.Condition, &dev)
	mongoExecute("device", func(col *mgo.Collection) {
		param := bson.M{}
		// var id string = conditon["id"].(string)
		id := dev.Id
		if id != "" {
			// param["id"] = ob.Id
			param["id"] = bson.M{"$regex": bson.RegEx{id, "i"}}

		}
		var result []Device
		col.Find(param).All(&result)
		count, _ := col.Find(param).Count()

		pr = &PageResult{1, 1, count, result}
	})

	return pr
}

func AddDevie(ob *Device) {
	mongoExecute("device", func(col *mgo.Collection) {
		err := col.Insert(ob)
		if err != nil {
			beego.Error("insert fail")
		}
	})
}

func UpdateDevice(ob *Device) {
	mongoExecute("device", func(col *mgo.Collection) {
		err := col.Update(bson.M{"id": ob.Id}, ob)
		if err != nil {
			beego.Error("insert fail")
		}
	})
}

func DeleteDevice(ob *Device) {
	mongoExecute("device", func(col *mgo.Collection) {
		err := col.Remove(bson.M{"id": ob.Id})
		if err != nil {
			beego.Error("insert fail")
		}
	})
}
