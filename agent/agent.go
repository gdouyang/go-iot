package agent

import (
	"fmt"
	"time"

	"encoding/json"

	"strings"

	"go-iot/provider/cmd"
	"go-iot/provider/ffmpeg"

	"github.com/astaxie/beego"
)

var (
	Client    cmd.Cli
	Processor []ffmpeg.C
	staStream []string
	queueName string
)

func init() {
	agentmode := beego.AppConfig.String("agentmode")
	if agentmode != "true" {
		return
	}
	uuid := "L1000000001"
	ip := "127.0.0.1"
	port := 19999
	beego.Info(fmt.Sprintf("agent init port:%d", port))
	go (func() {
		//TCP for ycds,south detect interface
		Client = cmd.Cli{Uid: uuid, Ip: ip, Port: port, Breath: 5, Mhandler: Handle}
		for {
			if Client.Dial() {
				//开启循环读取
				for {
					go Client.Read(Client.Mhandler)
					//发送ready
					if ok := Client.Ready(); ok != nil {
						fmt.Println("ready failed: ")
						fmt.Println(ok)
						Client.Close()
						break
					}
					time.Sleep(5 * time.Second)
				}
			}
			time.Sleep(5 * time.Second)
			fmt.Println("ygg 重连")
			continue
		}
	})()
}

func Handle(b []byte) {
	for {
		m := cmd.BytesToInt32(b[:4])
		//解析四位，如果长度大于原长度，认为解析错误，丢弃
		if m > uint32(len(b)) {
			break
		}
		n := b[4 : m+4]
		//调用解析
		fmt.Println(string(n))
		go func(bb []byte) {
			//			pp := lib.Pas{Request: {X: 0.0, Y: 0.0, Z: 0.0, SOnvif: "192.168.6.243:80", UserOnvif: "admin", PassOnvif: "admin"}}
			pp := Pas{}
			Parser(pp, bb)
		}(n)
		b = b[m+3 : len(b)]
		//嵌套，用于分离粘连
		if len(b) < 4 {
			break
		}
		Handle(b)
	}
}

func Parser(pmp Pas, b []byte) {
	if strings.Contains(string(b), "code") {
		var rr Result = Result{}
		json.Unmarshal(b, &rr)
	} else {
		//当有code，说明是返回，否则为请求
		json.Unmarshal(b, &pmp.Request)
		if strings.EqualFold(pmp.Action, "play") {
			//开启流
			fl := true
			for _, tst := range Processor {
				if strings.EqualFold(tst.Name, pmp.CameraCode) {
					fl = false //已经开启则不再操作
					break
				}
			}
			if fl {
				streamName := fmt.Sprintf("%s.flv", pmp.CameraCode)
				m, err := StreamLiveVideo(pmp.RtspUrl, queueName, streamName, pmp.CameraCode)
				if err != nil {
					//返回失败
					Client.RespFailed(pmp.Rspuuid, err)
				}
				//wait the stream for cache
				time.Sleep(3 * time.Second)
				Client.RespSuccess(pmp.Rspuuid)
				Processor = append(Processor, m)
				return
			}
			Client.RespSuccess(pmp.Rspuuid)
			return
		}
		if strings.EqualFold(pmp.Action, "stopTransform") {
			//停止转流
			fl := true
			for _, tst := range staStream {
				if strings.EqualFold(tst, pmp.CameraCode) {
					fl = false //不再操作
				}
			}
			if fl {
				for index, m := range Processor {
					//摄像机编码作为唯一标识
					if strings.EqualFold(m.Name, pmp.CameraCode) {
						ClosefmgVideo(m)
						Client.RespSuccess(pmp.Rspuuid)
						//delete one element from the slice
						Processor = append(Processor[:index], Processor[index+1:]...)
						return
					}
				}
			}
			Client.RespSuccess(pmp.Rspuuid)
			return
		}
		if strings.EqualFold(pmp.Action, "capture") {
			//抓图
			GetfmgFrame(pmp.RtspUrl, "./", fmt.Sprintf("%s%d", pmp.CameraCode, pmp.SendTime))
			time.Sleep(3 * time.Second)
			Client.RespPicSuccess(pmp.Rspuuid, fmt.Sprintf("%s%d", pmp.CameraCode, pmp.SendTime))
		}
		if strings.EqualFold(pmp.Action, "ptz") {
			//1.ptz controller
		}
		if strings.EqualFold(pmp.Action, "setPreset") {
			//1.ptz controller
		}
		if strings.EqualFold(pmp.Action, "goOver") {
			//1.through over the net
			//协议、ip、端口、参数、请求体
		}
		if strings.EqualFold(pmp.Action, "Ctrl") {
			//1.through over the net
		}
	}

}
