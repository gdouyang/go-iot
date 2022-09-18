package tcpserver

import (
	"bufio"
	"log"
	"net"
	"sync/atomic"

	"github.com/robertkrimen/otto"
)

type DelimType string

// 分隔符
const DelimType_Delimited DelimType = "Delimited"

// 固定长度
const DelimType_FixLength DelimType = "FixLength"

// 自定义拆分函数
const DelimType_SplitFunc DelimType = "SplitFunc"

func newDelimeter(delimeter TcpDelimeter, c net.Conn) Delimeter {
	var d Delimeter
	if delimeter.Type == DelimType_Delimited {
		b := []byte(delimeter.Delimited)
		d1 := &DelimeterDelimited{delim: b[len(b)-1], c: c}
		d1.init()
		d = d1
	} else if delimeter.Type == DelimType_FixLength {
		d1 := &DelimeterFixLength{buf: make([]byte, delimeter.Length), c: c}
		d1.init()
		d = d1
	} else if delimeter.Type == DelimType_SplitFunc {
		d1 := &PipePayloadParser{fun: delimeter.SplitFunc, c: c}
		d1.init()
		d = d1
	}
	return d
}

type Delimeter interface {
	Read() ([]byte, error)
}

type DelimeterDelimited struct {
	delim  byte // 分隔符
	c      net.Conn
	reader *bufio.Reader
}

func (d *DelimeterDelimited) init() {
	d.reader = bufio.NewReader(d.c)
}

func (d *DelimeterDelimited) Read() ([]byte, error) {
	data, err := d.reader.ReadSlice(d.delim)
	return data, err
}

// fix length
type DelimeterFixLength struct {
	buf    []byte // buf
	c      net.Conn
	reader *bufio.Reader
}

func (d *DelimeterFixLength) init() {
	d.reader = bufio.NewReader(d.c)
}

func (d *DelimeterFixLength) Read() ([]byte, error) {
	count, err := d.reader.Read(d.buf)
	data := d.buf[0:count]
	return data, err
}

// custom split func
// type DelimeterSplitFunc struct {
// 	fun     string
// 	c       net.Conn
// 	scanner *bufio.Scanner
// 	data    chan []byte
// 	buf     []byte
// }

// func (d *DelimeterSplitFunc) init() {
// 	d.data = make(chan []byte, 1)
// 	d.scanner = bufio.NewScanner(d.c)
// 	vm := otto.New()
// 	_, err := vm.Run(d.fun)
// 	if err != nil {
// 		log.Println(err)
// 	}
// 	splitFunc, _ := vm.Get("splitFunc")
// 	d.scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
// 		result, err := splitFunc.Call(splitFunc, data)
// 		if err != nil {
// 			log.Panicln(err)
// 		}
// 		if result.IsDefined() {
// 			obj := result.Object()
// 			nx, _ := obj.Get("next")
// 			num, _ := nx.ToInteger()
// 			advance = int(num)
// 			da, _ := obj.Get("result")
// 			str, _ := da.ToString()
// 			token = []byte(str)
// 			return advance, token, nil
// 		}

// 		if !atEOF {
// 			return 0, nil, nil
// 		}
// 		return 0, data, bufio.ErrFinalToken
// 	})
// 	go func() {
// 		for d.scanner.Scan() {
// 			bytes := d.scanner.Bytes()
// 			d.data <- bytes
// 		}
// 	}()
// }

// func (d *DelimeterSplitFunc) Read() (data []byte, err error) {
// 	return <-d.data, err
// }

type PipePayloadParser struct {
	c           net.Conn
	fun         string
	reader      *bufio.Reader
	pipe        []func(data []byte)
	result      []byte
	dataChan    chan []byte
	currentPipe atomic.Int32
}

func (p *PipePayloadParser) init() {
	p.dataChan = make(chan []byte, 1)
	p.reader = bufio.NewReader(p.c)

	vm := otto.New()
	_, err := vm.Run(p.fun)
	if err != nil {
		log.Panicln(err)
	}
	splitFunc, _ := vm.Get("splitFunc")
	_, err = splitFunc.Call(splitFunc, p)
	if err != nil {
		log.Panicln(err)
	}
}

func (p *PipePayloadParser) Delimited(delim string) *PipePayloadParser {
	go func() {
		for {
			b := []byte(delim)
			data, err := p.reader.ReadSlice(b[len(b)-1])
			if err != nil {
				log.Println(err)
			}
			p.AppendResult(data)

			handler := p.getNextHandler()
			if handler != nil {
				handler(data)
			}
		}
	}()
	return p
}

func (p *PipePayloadParser) Fixed(size int) {
	go func() {
		for {
			buf := make([]byte, size)
			count, err := p.reader.Read(buf)
			if err != nil {
				log.Println(err)
			}
			data := buf[0:count]
			p.AppendResult(data)

			handler := p.getNextHandler()
			if handler != nil {
				handler(data)
			}
		}
	}()
}

func (p *PipePayloadParser) AddHandler(handler func(data []byte)) {
	p.pipe = append(p.pipe, handler)
}

func (p *PipePayloadParser) Complete() {
	p.dataChan <- p.result
	p.currentPipe.Store(0)
	p.result = p.result[0:0]
}

func (p *PipePayloadParser) Read() (data []byte, err error) {
	return <-p.dataChan, err
}

func (p *PipePayloadParser) AppendResult(data []byte) {
	p.result = append(p.result, data...)
}

func (p *PipePayloadParser) getNextHandler() func([]byte) {
	if len(p.pipe) == 0 {
		return nil
	}
	index := p.currentPipe.Load()
	p.currentPipe.Add(1)
	if len(p.pipe) > int(index) {
		return p.pipe[index]
	} else {
		p.currentPipe.Store(0)
		return p.pipe[0]
	}
}
