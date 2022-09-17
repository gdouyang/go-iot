package tcpserver

import (
	"bufio"
	"net"
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
		d1 := &DelimeterSplitFunc{fun: delimitedFunc, c: c}
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
type DelimeterSplitFunc struct {
	fun     bufio.SplitFunc
	c       net.Conn
	scanner *bufio.Scanner
	data    chan []byte
}

func (d *DelimeterSplitFunc) init() {
	d.data = make(chan []byte, 1)
	d.scanner = bufio.NewScanner(d.c)
	d.scanner.Split(delimitedFunc)
	go func() {
		for d.scanner.Scan() {
			bytes := d.scanner.Bytes()
			d.data <- bytes
		}
	}()
}

func (d *DelimeterSplitFunc) Read() (data []byte, err error) {
	return <-d.data, err
}

func delimitedFunc(data []byte, atEOF bool) (advance int, token []byte, err error) {
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			return i+1, data[:i+1], nil
		}
	}
	if !atEOF {
		return 0, nil, nil
	}
	return 0, data, bufio.ErrFinalToken
}
