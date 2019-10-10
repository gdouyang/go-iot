/**
定义的接口，用于对接上位机的netty服务
**/
package cmd

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
)

type Handler func(b []byte)

var ready string

type Cli struct {
	Uid       string
	Transform string
	Conn      net.Conn
	Ip        string
	Port      int
	Breath    int
	Mhandler  Handler
}

type aut struct {
	Uuid          string `json:"uuid"`
	Action        string `json:"action"`
	TransformCode string `json:"transformCode"`
}

func (c *Cli) Dial() bool {
	ready = "{\"uuid\":\"ygg\",\"action\":\"init\",\"transformCode\":\"ZL00008\"}"
	tcpAddr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", c.Ip, c.Port))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
		return false
	}

	c.Conn, err = net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s\n", err.Error())
		return false
	}

	return true
}

func (c *Cli) Close() {
	c.Conn.Close()
}

//每5秒读取一次服务端的
func (c *Cli) Read(hand Handler) {
	var buffer = make([]byte, 4096)
	var tmp = make([]byte, 4096)
	for {
		m, err := c.Conn.Read(buffer)
		if err != nil && err != io.EOF {
			c.Conn.Close()
			break
		}
		if len(c.Conn.RemoteAddr().String()) < 0 {
			fmt.Println("we lose the server")
			c.Conn.Close()
			break
		}
		if m > 0 {
			hand(buffer)
		}
		buffer = tmp
	}
}

func (c *Cli) Write(b []byte) (err error) {
	_, err = c.Conn.Write(b)
	if err != nil {
		return err
	}
	return nil
}

func (c *Cli) Ready() (err error) {
	return c.Write(send(ready))
}

func intToBytes(n int) []byte {
	b := make([]byte, 4)
	b[3] = (byte)(n & 0xff)
	b[2] = (byte)(n >> 8 & 0xff)
	b[1] = (byte)(n >> 16 & 0xff)
	b[0] = (byte)(n >> 24 & 0xff)
	return b
}
func BytesToInt32(b []byte) uint32 {
	a := b[:4]
	return binary.BigEndian.Uint32(a)
}

func send(a string) (m []byte) {
	b := []byte(a)
	//	f := intToBytes(len(b))
	//	//	fmt.Println(f)
	//	n := bytes.NewBuffer(f)
	//	n.Write(b)
	//	//	fmt.Println(n.Bytes())
	//	m = hex.EncodeToString(n.Bytes())
	//	fmt.Println(m)
	//	return
	return EnPackSendData(b)
	//	return string(EnPackSendData(b))

}

//transform string to hex ,without crc check
func EnPackSendData(sendBytes []byte) []byte {
	packetLength := len(sendBytes) + 4
	result := make([]byte, packetLength)
	result[0] = byte(uint16(len(sendBytes)) >> 24)
	result[1] = byte(uint16(len(sendBytes)) >> 16)
	result[2] = byte(uint16(len(sendBytes)) >> 8)
	result[3] = byte(uint16(len(sendBytes)) & 0xFF)
	copy(result[4:], sendBytes)
	//	sendCrc := crc32.ChecksumIEEE(sendBytes)
	//	result[packetLength-4] = byte(sendCrc >> 24)
	//	result[packetLength-3] = byte(sendCrc >> 16 & 0xFF)
	//	result[packetLength-2] = 0xFF
	//	result[packetLength-1] = 0xFE
	return result
}

func (c *Cli) RespSuccess(uuid string) {
	mp := fmt.Sprintf("{\"uuid\":\"%s\",\"code\":0,\"msg\":\"success\"}", uuid)
	c.Write(send(mp))
}
func (c *Cli) RespPicSuccess(uuid string, path string) {
	//	mp := fmt.Sprintf("{\"uuid\":\"%s\",\"code\":0,\"msg\":\"success\"}", uuid)
	//	c.Write(send(mp))
	ff, _ := ioutil.ReadFile(fmt.Sprintf(".\\pic\\%s.jpg", path))
	abc := base64.StdEncoding.EncodeToString(ff)
	mp := fmt.Sprintf("{\"uuid\":\"%s\",\"code\":0,\"msg\":\"success\",\"data\":\"%s\"}", uuid, abc)
	c.Write(send(mp))
}
func (c *Cli) RespFailed(uuid string, err error) {
	mp := fmt.Sprintf("{\"uuid\":\"%s\",\"code\":0,\"msg\":\"%s\"}", uuid, err)
	c.Write(send(mp))
}
