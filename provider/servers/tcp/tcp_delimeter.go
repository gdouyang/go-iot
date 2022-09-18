package tcpserver

import (
	"bufio"
	"log"
	"math"
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

// 分隔符
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

// pipe
type PipePayloadParser struct {
	c           net.Conn
	fun         string
	pipe        []func(data []byte)
	result      []byte
	dataChan    chan []byte
	currentPipe atomic.Int32
	firstInit   func(parser *payloadParser)
	parser      *payloadParser
}

func (p *PipePayloadParser) init() {
	p.dataChan = make(chan []byte, 1)
	p.parser = newPayloadParser(bufio.NewReader(p.c))
	p.parser.handler = func(b []byte) {
		handler := p.getNextHandler()
		handler(b)
	}
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
	if !p.parser.started {
		p.firstInit = func(parser *payloadParser) {
			parser.delimitedMode(delim)
		}
		p.parser.handle()
	}
	p.parser.delimitedMode(delim)
	return p
}

func (p *PipePayloadParser) Fixed(size int) {
	if !p.parser.started {
		p.firstInit = func(parser *payloadParser) {
			parser.fixedSizeMode(size)
		}
		p.parser.handle()
	}
	p.parser.fixedSizeMode(size)
}

func (p *PipePayloadParser) AddHandler(handler func(data []byte)) {
	p.pipe = append(p.pipe, handler)
}

func (p *PipePayloadParser) Complete() {
	p.dataChan <- p.result
	p.currentPipe.Store(0)
	p.result = p.result[0:0]
	if p.firstInit != nil {
		p.firstInit(p.parser)
	}
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

// new payloadParser func
func newPayloadParser(reader *bufio.Reader) *payloadParser {
	return &payloadParser{reader: reader, demand: math.MaxInt64}
}

// payloadParser see vertx RecordParserImpl
type payloadParser struct {
	reader        *bufio.Reader
	delimited     bool // mode of delimited
	delim         []byte
	started       bool
	buff          []byte
	handler       func([]byte)
	parsing       bool
	pos           int // Current position in buffer
	start         int // Position of beginning of current record
	delimPos      int // Position of current match in delimiter array
	recordSize    int
	maxRecordSize int
	demand        int64
}

func (p *payloadParser) delimitedMode(delim string) {
	p.started = true
	p.delimited = true
	p.delim = []byte(delim)
	p.delimPos = 0
}
func (p *payloadParser) fixedSizeMode(size int) {
	p.started = true
	p.delimited = false
	p.recordSize = size
}

func (p *payloadParser) handle() {
	go func() {
		for {
			buf := make([]byte, 100)
			count, err := p.reader.Read(buf)
			if err != nil {
				log.Println(err)
			}
			data := buf[0:count]
			p.buff = append(p.buff, data...)
			p.handleParsing()
			if p.buff != nil && p.maxRecordSize > 0 && len(p.buff) > p.maxRecordSize {
				log.Fatalln("The current record is too long")
			}
		}
	}()
}

func (p *payloadParser) handleParsing() {
	if p.parsing {
		return
	}
	p.parsing = true
	defer func() { p.parsing = false }()
	for {
		if p.demand > 0 {
			var next int
			if p.delimited {
				next = p.parseDelimited()
			} else {
				next = p.parseFixed()
			}
			if next == -1 {
				break
			}
			if p.demand != math.MaxInt64 {
				p.demand--
			}
			data := p.buff[p.start:next]
			p.start = p.pos
			if p.handler != nil {
				p.handler(data)
			}
		} else {
			break
		}
	}
	length := len(p.buff)
	if p.start == length {
		p.buff = make([]byte, 0)
	} else if p.start > 0 {
		p.buff = p.buff[p.start:length]
	}
	p.pos -= p.start
	p.start = 0
}

func (p *payloadParser) parseDelimited() int {
	length := len(p.buff)
	for ; p.pos < length; p.pos++ {
		if p.buff[p.pos] == p.delim[p.delimPos] {
			p.delimPos++
			if p.delimPos == len(p.delim) {
				p.pos++
				p.delimPos = 0
				return p.pos - len(p.delim)
			}
		} else {
			if p.delimPos > 0 {
				p.pos -= p.delimPos
				p.delimPos = 0
			}
		}
	}
	return -1
}

func (p *payloadParser) parseFixed() int {
	len := len(p.buff)
	if len-p.start >= p.recordSize {
		end := p.start + p.recordSize
		p.pos = end
		return end
	}
	return -1
}
