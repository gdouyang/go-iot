package modelfactory

import (
	"go-iot/models/led"
	"go-iot/models/operates"
)

func GetDevice(id string) (operates.Device, error) {
	var dev operates.Device
	l, err := led.GetDevice(id)
	if err != nil {
		return dev, err
	}
	dev = operates.Device{Id: l.Id, Sn: l.Sn, Name: l.Name, Provider: l.Provider}
	return dev, nil
}
