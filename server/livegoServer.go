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

// var HlsListen, RtmpListen, FlvListen net.Listener
var err error

type LiveMedia struct {
	HlsListen  net.Listener
	RtmpListen net.Listener
	FlvListen  net.Listener
}

func NEW() *LiveMedia {
	return new(LiveMedia)
}

func (this *LiveMedia) Start(srs string) {
	// 配置
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
			if this.RtmpListen == nil {
				beego.Info("RTMP server listen address : ", sa.Port)
				if stream == nil {
					stream = rtmp.NewRtmpStream()
				}
				if hlserver != nil {
					this.startRtmp(stream, hlserver, fmt.Sprintf(":%d", sa.Port))
					continue
				}
				this.startRtmp(stream, nil, fmt.Sprintf(":%d", sa.Port))
			}
		}
		if strings.EqualFold("flv", sa.Type) {
			if this.FlvListen == nil {
				beego.Info("HTTP-FLV server listen address : ", sa.Port)
				if stream == nil {
					stream = rtmp.NewRtmpStream()
				}
				this.startHTTPFlv(stream, fmt.Sprintf(":%d", sa.Port))
			}
		}
		if strings.EqualFold("hls", sa.Type) {
			if this.HlsListen == nil {
				beego.Info("HLS server listen address : ", sa.Port)
				if stream == nil {
					stream = rtmp.NewRtmpStream()
				}
				this.startHls(fmt.Sprintf(":%d", sa.Port))
			}
		}
	}
}

/*启动恢复*/
func (this *LiveMedia) ResumeAll() {
	// 配置
	ss, err := camera.GetServerAllStatus()
	if err != nil {
		beego.Error(err)
		return
	}
	for _, sa := range ss {
		if strings.EqualFold("on", sa.Status) {
			if strings.EqualFold("rtmp", sa.Type) {
				beego.Info("RTMP server listen address : ", sa.Port)
				if stream == nil {
					stream = rtmp.NewRtmpStream()
				}
				if hlserver != nil {
					this.startRtmp(stream, hlserver, fmt.Sprintf(":%d", sa.Port))
					continue
				}
				this.startRtmp(stream, nil, fmt.Sprintf(":%d", sa.Port))
			}
			if strings.EqualFold("flv", sa.Type) {
				beego.Info("HTTP-FLV server listen address : ", sa.Port)
				if stream == nil {
					stream = rtmp.NewRtmpStream()
				}
				this.startHTTPFlv(stream, fmt.Sprintf(":%d", sa.Port))
			}
			if strings.EqualFold("hls", sa.Type) {
				beego.Info("HLS server listen address : ", sa.Port)
				if stream == nil {
					stream = rtmp.NewRtmpStream()
				}
				this.startHls(fmt.Sprintf(":%d", sa.Port))
			}
		}
	}
}

func (this *LiveMedia) StopAll() error {
	err := this.stop("hls")
	if err != nil {
		return err
	}
	camera.SetServerStatus("off", "hls")
	err = this.stop("rtmp")
	if err != nil {
		return err
	}
	camera.SetServerStatus("off", "rtmp")
	err = this.stop("flv")
	if err != nil {
		return err
	}
	camera.SetServerStatus("off", "flv")
	return nil
}

func (this *LiveMedia) startHls(addr string) *hls.Server {
	this.HlsListen, err = net.Listen("tcp", addr)
	if err != nil {
		beego.Error(err)
		this.HlsListen = nil
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
	go hlsServer.Serve(this.HlsListen)
	// update live status
	return hlsServer
}

func (this *LiveMedia) startRtmp(stream *rtmp.RtmpStream, hlsServer *hls.Server, addr string) {
	this.RtmpListen, err = net.Listen("tcp", addr)
	if err != nil {
		beego.Error(err)
		this.RtmpListen = nil
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
	go rtmpServer.Serve(this.RtmpListen)
}

func (this *LiveMedia) startHTTPFlv(stream *rtmp.RtmpStream, addr string) {
	this.FlvListen, err = net.Listen("tcp", addr)
	if err != nil {
		beego.Error(err)
		this.FlvListen = nil
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
	go hdlServer.Serve(this.FlvListen)
}

func (this *LiveMedia) stop(abc string) error {
	if strings.EqualFold(abc, "rtmp") {
		if this.RtmpListen == nil {
			beego.Info("rtmp stoped ever!")
			return nil
		}
		err := this.RtmpListen.Close()
		if err != nil {
			return err
		}
		this.RtmpListen = nil
		return nil
	}
	if strings.EqualFold(abc, "flv") {
		if this.FlvListen == nil {
			beego.Info("flv stoped ever!")
			return nil
		}
		err := this.FlvListen.Close()
		if err != nil {
			return err
		}
		this.FlvListen = nil
		return nil
	}
	if strings.EqualFold(abc, "hls") {
		if this.HlsListen == nil {
			beego.Info("hls stoped ever!")
			return nil
		}
		err := this.HlsListen.Close()
		if err != nil {
			return err
		}
		this.HlsListen = nil
		return nil
	}
	return nil
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
