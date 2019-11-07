package agent

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"go-iot/models"
	"go-iot/models/modelfactory"
	"go-iot/models/operates"
	"go-iot/provider/util"

	"github.com/astaxie/beego"
	"github.com/gorilla/websocket"
)

func init() {
	beego.Router("/agetn/ws", &AgentWebSocket{}, "get:Join")
}

type AgentClient struct {
	uid      int             //WebSocket连接唯一标识
	SN       string          //设备SN
	Conn     *websocket.Conn // websocket连接
	Cond     *sync.Cond      // 同步调用的condition
	respChan chan int        // 命令响应Channel
	Resp     models.JsonResp // 命令返回
}

// Agent消息请求
type AgentRequest struct {
	DeviceId string          // 设备Id
	SN       string          // 设备SN
	Provider string          // 厂商
	Oper     string          // 操作标识
	Data     json.RawMessage // 请求数据
}

func NewRequest(devId, sn, provider, oper string, data json.RawMessage) AgentRequest {
	return AgentRequest{DeviceId: devId, SN: sn, Provider: provider, Oper: oper, Data: data}
}

// 心跳
type breath struct {
	Sn string `json:"sn"`
}

// AgentWebSocket
type AgentWebSocket struct {
	beego.Controller
}

var (
	subscribers        = map[string]*AgentClient{}
	providerId  string = "agent"
)

// 加入方法
func (this *AgentWebSocket) Join() {
	// Upgrade from http request to WebSocket.
	c, err := websocket.Upgrade(this.Ctx.ResponseWriter, this.Ctx.Request, nil, 1024, 1024)
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(this.Ctx.ResponseWriter, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		beego.Error("Cannot setup WebSocket connection:", err)
		return
	}

	var sn string

	var l sync.Mutex
	agent := &AgentClient{uid: util.Uuid(), SN: sn, Conn: c, Cond: sync.NewCond(&l), respChan: make(chan int, 2)}
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			beego.Error("ws read err:", err)
			break
		}
		resp := string(message)
		abc := breath{}
		json.Unmarshal(message, &abc)
		sn = abc.Sn
		if len(sn) > 0 {
			beego.Info("agent breath -----> ", sn, "mssageType:", mt, "message :", resp)
			_, ok := subscribers[sn]
			if !ok {
				agent.SN = sn
				subscribers[sn] = agent
				evt := operates.DeviceOnlineStatus{OnlineStatus: models.ONLINE, Sn: sn, Type: providerId}
				modelfactory.FireOnlineStatus(evt)
				beego.Info("agent connected, len:", len(subscribers))
			}
		} else {
			l, ok := subscribers[agent.SN]
			if ok {
				beego.Info("agent response -----> ", agent.SN, "mssageType:", mt, "message :", resp)
				l.Cond.L.Lock()
				aResp := models.JsonResp{}
				json.Unmarshal(message, &aResp)
				l.Resp = aResp // 返回响应消息
				l.Cond.Signal()
				l.Cond.L.Unlock()
				l.respChan <- 1
			} else {
				beego.Warn("not found connection sn:", agent.SN)
			}
		}
	}
	sn = agent.SN
	defer func() {
		beego.Info("agent close sn:", sn)
		obj, ok := subscribers[sn]
		if ok {
			obj.Conn.Close()
			delete(subscribers, sn)
		} else {
			c.Close()
		}
		evt := operates.DeviceOnlineStatus{OnlineStatus: models.OFFLINE, Sn: sn, Type: providerId}
		modelfactory.FireOnlineStatus(evt)
	}()
}

// 发送命令给Agent，并等待响应
func SendCommand(sn string, command AgentRequest) models.JsonResp {
	agent, ok := subscribers[sn]
	if ok {
		data, err := util.JsonEncoderHTML(command)
		if err != nil {
			return models.JsonResp{Success: false, Msg: err.Error()}
		}
		// LED没有返回的情况需要处理超时
		go checkTimeout(agent)
		// 把当前请求等待,等待接口返回
		agent.Cond.L.Lock()
		beego.Info("agent send command", command)
		agent.Conn.WriteMessage(websocket.BinaryMessage, data)
		agent.Cond.Wait()
		agent.Cond.L.Unlock()
		beego.Info("agent.Resp", agent.Resp)
		return agent.Resp
	}
	return models.JsonResp{Success: false, Msg: sn + "没有在线"}
}

// 没有返回的情况需要处理超时
func checkTimeout(agent *AgentClient) {
	select {
	case <-agent.respChan:
		beego.Info("send command success resp")
	case <-time.Tick(time.Second * 20):
		agent.Cond.L.Lock()
		beego.Info("send command has timeout")
		agent.Resp = models.JsonResp{Success: false, Msg: "agent timeout"}
		agent.Cond.Signal()
		agent.Cond.L.Unlock()
	}
}
