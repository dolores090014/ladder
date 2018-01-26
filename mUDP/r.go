package mUDP

import (
	"time"
	"context"
	"fmt"
	"net/http"
	"net"
	"bufio"
	"bytes"
	"strconv"
)

type Carrier struct {
	window    *window
	timePoint time.Time
	ch        chan *Protocol
	ctx       context.Context
	cancel    context.CancelFunc
	buffer    *SimpleRingBuffer
	//requestBin        []byte
	requestId         int
	webSiteTCPConnect *net.TCPConn
	requestInfo       *http.Request
}

func NewRequest(id int, bin []byte) {
	request, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(bin)))
	if err != nil {
		return
	}
	TunnelRemote.Lock()
	if v, ok := TunnelRemote.r[id]; ok {
		v.webSiteTCPConnect.Write(bin)
	} else {
		c := NewCarrier()
		c.requestId = id
		TunnelRemote.r[id] = c
		ip, _ := net.LookupIP(request.Host)
		//_ = ip
		//ip := [][]byte{[]byte("172.19.3.65")} //todo remove
		port := 80
		if request.Method == "CONNECT" {
			port = 443
		}
		adr2, err := net.ResolveTCPAddr("tcp", ip[0].String()+":"+strconv.Itoa(port))
		c.webSiteTCPConnect, err = net.DialTCP("tcp", nil, adr2)
		if err != nil && port == 443 {
			c.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
		}

		go c.getDataFromCarrier(c.ctx)
		go c.transfer(c.ctx)
		go c.read(c.ctx)
		c.webSiteTCPConnect.Write(bin)
	}
	TunnelRemote.Unlock()
}

func CloseRequest(id int) {

}

func NewCarrier() *Carrier {
	o := &Carrier{
		window: &window{
			value: 1024,
		},
		ch: make(chan *Protocol, 1024),
	}
	o.ctx, o.cancel = context.WithCancel(context.Background())
	o.buffer = NewSimpleRingBuffer(1024 * 8)
	o.buffer.AutoUpdateStartPoint(false)
	return o
}

func (this *Carrier) Close() {
	defer func() {
		if recover() != nil {
			return
		}
	}()
	this.webSiteTCPConnect.Close()
	fmt.Println("remote close")
	//close(this.chanForDataBack)
	this.cancel()
}

func (this *Carrier) getDataFromCarrier(ctx context.Context) {
	for {
		select {
		case <-this.ctx.Done():
		default:
			bin := make([]byte, 1024)
			n, err := this.webSiteTCPConnect.Read(bin)
			if err != nil {
				this.Close()
				return
			}
			err = this.Write(bin[:n])
			if err != nil {
				fmt.Println("upd channel close")
				return
			}
		}
		//this.chanForDataBack <- bin[:n]
	}
}

//get pack again and push to channel
func (this *Carrier) sendAgain(offset int) {
	el := this.buffer.ReadOffset(offset)
	el.offset = uint32(offset)
	TunnelRemote.PriorWrite(el.EncodeDataPack(), true)
}

//write to connect

//write to buffer
func (this *Carrier) Write(b []byte) error {
	lenth := 0
	for {
		el := NewProtocol()
		el.act = PROTOCOL_DATA
		el.requestId = uint16(this.requestId)
		if lenth == 0 {
			lenth = len(el.data)
		}
		if len(b) <= lenth {
			copy(el.data, b)
			el.data = el.data[:len(b)]
			this.buffer.Write(el)
			break
		} else {
			copy(el.data, b[:lenth])
			b = b[lenth:]
			this.buffer.Write(el)
		}
	}
	return nil
}

// from buffer to channel
func (this *Carrier) transfer(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			time.Sleep(100 * time.Millisecond)
			bin, n, start := this.buffer.Read(int(this.window.window()))
			if n == 0 {
				continue
			}
			bin = bin[:n]
			for k, v := range bin {
				el := (*Protocol)(v)
				el.windowStart = uint32(start)
				el.offset = uint32(start + k)
				TunnelRemote.PriorWrite(el.EncodeDataPack(), false)
			}
		}
	}
}

//read from connect
func (this *Carrier) read(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			el := <-this.ch
			switch el.act {
			case PROTOCOL_HEART_BEATS:
				this.buffer.start = int(el.windowStart)
			case PROTOCOL_SERIAL:
				for _, v := range el.missList {
					go this.sendAgain(int(v))
				}
			case PROTOCOL_WINDOW:
				this.window.speedControll(float32(el.wr / 100))
			case PROTOCOL_END:
				fmt.Println("r.go close")
				this.Close()
				return
			}
		}
	}
}
