package server

import (
	"fmt"
	"go-iot/models/camera"
	"net"
	"strings"

	"github.com/astaxie/beego"
	"github.com/gwuhaolin/livego/protocol/hls"
	"github.com/gwuhaolin/livego/protocol/httpflv"
	"github.com/gwuhaolin/livego/protocol/rtmp"
)

var stream = new(rtmp.RtmpStream)
var hlserver = new(hls.Server)
var hlsListen, rtmpListen, flvListen net.Listener
var err error

func init() {
	First_StartAll()
}

func CheckMediaServer() {
	for {
		beego.Info("用于监控状态，展示不实现")
	}
}

func Start(srs string) {
	ss, err := camera.GetServerAllStatus()
	if !strings.EqualFold("all", srs) {
		ss, err = camera.GetServerStatus(srs)
	}
	if err != nil {
		beego.Error(err)
		return
	}
	for _, sa := range ss {
		if strings.EqualFold("rtmp", sa.Type) {
			beego.Info("RTMP server listen address : ", sa.Port)
			if stream == nil {
				stream = rtmp.NewRtmpStream()
			}
			if hlserver != nil {
				go startRtmp(stream, hlserver, fmt.Sprintf(":%d", sa.Port))
				continue
			}
			go startRtmp(stream, nil, fmt.Sprintf(":%d", sa.Port))
		}
		if strings.EqualFold("http_flv", sa.Type) {
			beego.Info("HTTP-FLV server listen address : ", sa.Port)
			if stream == nil {
				stream = rtmp.NewRtmpStream()
			}
			go startHTTPFlv(stream, fmt.Sprintf(":%d", sa.Port))
		}
		if strings.EqualFold("hls", sa.Type) {
			beego.Info("HLS server listen address : ", sa.Port)
			if stream == nil {
				stream = rtmp.NewRtmpStream()
			}
			go startHls(fmt.Sprintf(":%d", sa.Port))
		}
	}
}

func First_StartAll() {
	ss, err := camera.GetServerAllStatus()
	if err != nil {
		beego.Error(err)
		return
	}
	for _, sa := range ss {
		if strings.EqualFold("on", sa.Status) {
			if rtmpListen != nil { // do not re listen
				if strings.EqualFold("rtmp", sa.Type) {
					beego.Info("RTMP server listen address : ", sa.Port)
					if stream == nil {
						stream = rtmp.NewRtmpStream()
					}
					if hlserver != nil {
						startRtmp(stream, hlserver, fmt.Sprintf(":%d", sa.Port))
						continue
					}
					go startRtmp(stream, nil, fmt.Sprintf(":%d", sa.Port))
				}
			}
			if flvListen != nil {
				if strings.EqualFold("http_flv", sa.Type) {
					beego.Info("HTTP-FLV server listen address : ", sa.Port)
					if stream == nil {
						stream = rtmp.NewRtmpStream()
					}
					go startHTTPFlv(stream, fmt.Sprintf(":%d", sa.Port))
				}
			}
			if hlsListen != nil {
				if strings.EqualFold("hls", sa.Type) {
					beego.Info("HLS server listen address : ", sa.Port)
					if stream == nil {
						stream = rtmp.NewRtmpStream()
					}
					go startHls(fmt.Sprintf(":%d", sa.Port))
				}
			}
		}
	}
}

func StopAll() error {
	err := stopHls()
	if err != nil {
		return err
	}
	err = stopRtmp()
	if err != nil {
		return err
	}
	err = stopHTTPFlv()
	if err != nil {
		return err
	}
	return nil
}

func startHls(addr string) *hls.Server {
	hlsListen, err = net.Listen("tcp", addr)
	if err != nil {
		beego.Error(err)
		hlsListen = nil
		return nil
	}

	hlsServer := hls.NewServer()
	defer func() {
		if r := recover(); r != nil {
			beego.Error("HLS server panic: ", r)
		}
	}()
	beego.Error("HLS listen On", addr)
	camera.SetServerStatus("on", "hls")
	hlsServer.Serve(hlsListen)
	// update live status
	return hlsServer
}

func stopHls() error {
	if hlsListen == nil {
		beego.Info("stoped ever")
		return nil
	}
	err := hlsListen.Close()
	if err != nil {
		return err
	}
	hlsListen = nil
	return camera.SetServerStatus("off", "hls")
}

func startRtmp(stream *rtmp.RtmpStream, hlsServer *hls.Server, addr string) {
	rtmpListen, err := net.Listen("tcp", addr)
	if err != nil {
		beego.Error(err)
		rtmpListen = nil
		return
	}

	var rtmpServer *rtmp.Server

	if hlsServer == nil {
		rtmpServer = rtmp.NewRtmpServer(stream, nil)
		beego.Info("hls server disable....")
	} else {
		rtmpServer = rtmp.NewRtmpServer(stream, hlsServer)
		beego.Info("hls server enable....")
	}

	defer func() {
		if r := recover(); r != nil {
			beego.Error("RTMP server panic: ", r)
		}
	}()
	beego.Info("RTMP Listen On", addr)
	camera.SetServerStatus("on", "rtmp")
	rtmpServer.Serve(rtmpListen)
}

func stopRtmp() error {
	if rtmpListen == nil {
		beego.Info("stoped ever")
		return nil
	}
	err := rtmpListen.Close()
	if err != nil {
		beego.Error(err)
		return err
	}
	rtmpListen = nil
	return camera.SetServerStatus("off", "rtmp")
}

func startHTTPFlv(stream *rtmp.RtmpStream, addr string) {
	flvListen, err := net.Listen("tcp", addr)
	if err != nil {
		beego.Error(err)
		flvListen = nil
		return
	}

	hdlServer := httpflv.NewServer(stream)
	defer func() {
		if r := recover(); r != nil {
			beego.Error("HTTP-FLV server panic: ", r)
			camera.SetServerStatus("off", "flv")
		}
	}()
	beego.Info("HTTP-FLV listen On", addr)
	camera.SetServerStatus("on", "flv")
	hdlServer.Serve(flvListen)
}

func stopHTTPFlv() error {
	if flvListen == nil {
		beego.Info("stoped ever")
		return nil
	}
	err := flvListen.Close()
	if err != nil {
		beego.Error(err)
		return err
	}
	flvListen = nil
	return camera.SetServerStatus("off", "flv")
}

//func startHTTPOpera(stream *rtmp.RtmpStream) {
//	if *operaAddr != "" {
//		opListen, err := net.Listen("tcp", *operaAddr)
//		if err != nil {
//			beego.Error(err)
//		}
//		opServer := httpopera.NewServer(stream, *rtmpAddr)
//		go func() {
//			defer func() {
//				if r := recover(); r != nil {
//					beego.Error("HTTP-Operation server panic: ", r)
//				}
//			}()
//			beego.Info("HTTP-Operation listen On", *operaAddr)
//			opServer.Serve(opListen)
//		}()
//	}
//}

//func main() {
//	defer func() {
//		if r := recover(); r != nil {
//			beego.Error("livego panic: ", r)
//			time.Sleep(1 * time.Second)
//		}
//	}()
//	err := configure.LoadConfig(*configfilename)
//	if err != nil {
//		return
//	}

//	stream := rtmp.NewRtmpStream()
//	hlsServer := startHls()
//	startHTTPFlv(stream)
//	startHTTPOpera(stream)

//	startRtmp(stream, hlsServer)
//	//startRtmp(stream, nil)
//}
