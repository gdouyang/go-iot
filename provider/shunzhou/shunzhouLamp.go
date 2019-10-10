package shunzhou

import (
	"go-iot/provider/utils"
)

type Shuncom_Lamp struct {
	Head byte
	Len  byte
	Addr []byte
	Port byte
	Act  byte
}

var (
	Head_Lamp_Get      byte = 0x01
	Head_Lamp_Ctl      byte = 0x15
	Head_Alarm_Cur_Set byte = 0x22
	Head_Energy_Get    byte = 0x25
	Head_GroupId_Get   byte = 0x36
	Head_Group_Set     byte = 0x37
	Head_Group_Get     byte = 0x38
	Head_Group_Ctrl    byte = 0x39
	Head_Alarm_Event        = [2]byte{0xAA, 0X5A}

	Port_Alarm     byte = 0x01
	Port_Regulate  byte = 0x04
	Port_OpenClose byte = 0x05

	Act_Close_Code byte = 0x01
	Act_Open_Code  byte = 0x00

	Upper_Limit_Code byte = 0x31
	Low_Limit_Code   byte = 0x32

	Alarm_Type_BlewCurrent byte = 0x81
	Alarm_Type_OverCurrent byte = 0x82
	Alarm_Type_BlewVoltage byte = 0x83
	Alarm_Type_OverVoltage byte = 0x84
)

func (ms *Shuncom_Lamp) Parse(src []byte) {
	if src[0] == 0xAA && src[1] == 0x5A {
		//告警
	}
}

func (ms *Shuncom_Lamp) Open() []byte {
	ms.Head = Head_Lamp_Ctl
	ms.Port = Port_OpenClose
	ms.Act = Act_Open_Code
	ms.Len = 0x0A
	var mp []byte
	mp = append(mp, ms.Head, ms.Len)
	mp = append(mp, ms.Addr...)
	mp = append(mp, ms.Port, ms.Act)
	ma := utils.CheckSum(mp)
	mp = append(mp, ma[0], ma[1])
	return mp
}

func (ms *Shuncom_Lamp) Close() []byte {
	ms.Head = Head_Lamp_Ctl
	ms.Port = Port_OpenClose
	ms.Act = Act_Close_Code
	ms.Len = 0x0A
	var mp []byte
	mp = append(mp, ms.Head, ms.Len)
	mp = append(mp, ms.Addr...)
	mp = append(mp, ms.Port, ms.Act)
	ma := utils.CheckSum(mp)
	mp = append(mp, ma[0], ma[1])
	return mp
}

func (ms *Shuncom_Lamp) Regulate() []byte {
	ms.Head = Head_Lamp_Ctl
	ms.Port = Port_Regulate
	ms.Act = Act_Close_Code
	ms.Len = 0x0A
	var mp []byte
	mp = append(mp, ms.Head, ms.Len)
	mp = append(mp, ms.Addr...)
	mp = append(mp, ms.Port, ms.Act)
	ma := utils.CheckSum(mp)
	mp = append(mp, ma[0], ma[1])
	return mp
}

//func (ms *Shuncom_Lamp) GetEnv() []byte {

//}
