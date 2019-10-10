package shunzhou

import (
	"encoding/hex"
	"fmt"
	"net"
	"strings"

	"github.com/astaxie/beego"
)

type Shuncom_Gateway struct {
	Addr          string
	Heart_Package string
	Conn          net.Conn
}

var ShunList []Shuncom_Gateway

// 收到心跳，设备上线，两分钟没有再次收到心跳的时候，设备离线
func init() {
	port := beego.AppConfig.DefaultInt("shunzhouport", 7077)
	beego.Info(fmt.Sprintf("shunzhou init port:%d", port))
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", "0.0.0.0", port))
	if err != nil {
		fmt.Printf("listen fail, err: %v\n", err)
		return
	}

	go (func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				fmt.Printf("accept fail, err: %v\n", err)
				continue
			}
			go process(conn)
		}
	})()
}

func process(conn net.Conn) {
	defer conn.Close()
	for {
		var buf [256]byte
		n, err := conn.Read(buf[:])

		if err != nil {
			fmt.Printf("read from connect failed, err: %v\n", err)
			break
		}
		str := hex.EncodeToString(buf[:n])
		str1 := string(buf[:n])
		fmt.Printf("receive from client, data: %v\n", str1)
		//判断是否心跳，如果是心跳，那么将验证是否已经在列表中，如果不在列表中，则添加，否则激活下
		if len(str) == 10 {
			f2 := false //是否已经存在
			for _, m := range ShunList {
				if strings.EqualFold(m.Heart_Package, str) {
					f2 = true
				}
			}
			if !f2 { //不存在添加该链接 dc412260bb 00240802
				ShunList = append(ShunList, Shuncom_Gateway{
					Addr:          "00248811",
					Heart_Package: "dc41226774",
					Conn:          conn})
			}
		}
	}
}

func (c *Shuncom_Gateway) OpenLamp(addr []byte) (status bool) {
	if len(addr) != 4 {
		return false
	}
	sl := Shuncom_Lamp{Addr: addr[0:4]}
	fmt.Println(hex.EncodeToString(sl.Open()))
	c.Conn.Write(sl.Open())
	return true
}

func (c *Shuncom_Gateway) CloseLamp(addr []byte) (status bool) {
	if len(addr) != 4 {
		return false
	}
	sl := Shuncom_Lamp{Addr: addr[0:4]}
	fmt.Println(hex.EncodeToString(sl.Close()))
	c.Conn.Write(sl.Close())
	return true
}

func (c *Shuncom_Gateway) RegulatorLamp(addr []byte) (status bool) {
	if len(addr) != 4 {
		return false
	}
	sl := Shuncom_Lamp{Addr: addr[0:4]}
	fmt.Println(hex.EncodeToString(sl.Open()))
	c.Conn.Write(sl.Close())
	return true
}
