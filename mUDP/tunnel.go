package mUDP

import (
	"net"
	"fmt"
	"time"
	"context"
	"strconv"
	"bytes"
	"encoding/binary"
	"sync"
	"math"
)

type Tunnel struct {
	ctx         context.Context
	cancel      context.CancelFunc
	able        bool
	delay       int16 //ms
	ackList     map[uint16]*Protocol
	priorChan   chan []byte
	normalChan  chan []byte
	readChan    chan *Protocol
	writeBuffer *SimpleRingBuffer
	readBuffer  *RingBufferWithMiss
	remote      *net.UDPAddr
	window      *window
	conn        *net.UDPConn
}

func NewTunnel(c bool) *Tunnel {
	ctx, canncel := context.WithCancel(context.Background())
	if c {
		return &Tunnel{
			ctx:        ctx,
			cancel:     canncel,
			readBuffer: NewWriteBuffer(1024 * 32),
			ackList:    make(map[uint16]*Protocol),
			readChan:   make(chan *Protocol, 128),
		}
	} else {
		return &Tunnel{
			ctx:    ctx,
			cancel: canncel,
			window: &window{
				value: 64,
			},
			writeBuffer: NewSimpleRingBuffer(1024 * 64),
			priorChan:   make(chan []byte, 1024*16),
			normalChan:  make(chan []byte, 1024*32),
		}
	}
}

/*
from buffer to connect
 */
func (this *Tunnel) Send() {
	for {
		var bin []byte
		select {
		case <-this.ctx.Done():
			close(this.priorChan)
			close(this.normalChan)
			return
		case data, ok := <-this.priorChan:
			if !ok {
				return
			}
			bin = data
		default:
			select {
			case data, ok := <-this.priorChan:
				if !ok {
					return
				}
				bin = data
			case data, ok := <-this.normalChan:
				if !ok {
					return
				}
				bin = data
			}
		}
		this.conn.WriteToUDP(bin, this.remote)
	}
}

/*
write buffer to channel
 */
func (this *Tunnel) send() {
	for {
		select {
		case <-this.ctx.Done():
			return
		default:
			time.Sleep(100 * time.Millisecond)
			bin, n, start := this.writeBuffer.Read(int(this.window.window()))

			if n == 0 {
				continue
			}
			bin = bin[:n]
			for k, v := range bin {
				el := (*Protocol)(v)
				el.windowStart = uint32(start)
				el.offset = uint32(start + k)
				this.normalChan <- el.EncodeDataPack()
			}
		}
	}
}

func (this *Tunnel) Able() bool {
	return this.able
}

func (this *Tunnel) heartBeats(ctx context.Context) {
	for {
		time.Sleep(time.Duration(HEART_BEAT_INTERVAL) * time.Second)
		select {
		case <-ctx.Done():
			return
		default:
			p := NewProtocol()
			p.Act = PROTOCOL_HEART_BEATS
			this.conn.Write(p.EncodeHeartBeatPack())
			//p.windowStart = uint32(this.buffer.start)
			//this.sendChan <- p.EncodeHeartBeatPack()
		}
	}
}

/*
send msg to remote
 */
func (this *Tunnel) Write(bin []byte) {
	this.conn.Write(bin)
}

func (this *Tunnel) SendMsgToLocal(id uint16, b []byte) error {
	lenth := 0
	for {
		el := NewProtocol()
		el.Act = PROTOCOL_DATA
		el.RequestId = uint16(id)
		if lenth == 0 {
			lenth = len(el.data)
		}
		if len(b) <= lenth {
			copy(el.data, b)
			el.data = el.data[:len(b)]
			this.writeBuffer.Write(el)
			break
		} else {
			copy(el.data, b[:lenth])
			b = b[lenth:]
			this.writeBuffer.Write(el)
		}
	}
	return nil
}

/*
local receive
*/
func (this *Tunnel) ReceiveMsgFromRemote() {
	for {
		var bin = make([]byte, UDP_SIZE)
		n, err := this.conn.Read(bin)
		if err != nil {
			fmt.Println(err)
		}
		bin = bin[:n]
		el := NewProtocol().Decode(bin)
		defer func() {
			if recover() != nil {
				fmt.Println(el.RequestId)
			}
		}()
		fmt.Println("write buffer",el.RequestId)
		this.readBuffer.Write(el, int(el.offset))
		for {
			ele := this.readBuffer.Read()
			fmt.Println("read buffer",ele.RequestId)
			if ele == nil {
				break
			}
			this.readChan <- ele
		}

		if el.offset%64 == 0 { //频率限制
			miss := this.readBuffer.Miss(int(el.windowStart)) //丢包
			this.GetMissPackAndRequestAgain(miss)
		}
	}
}

func (this *Tunnel) ReceiveMsgFromLocal() {
	once := &sync.Once{}
	for {
		var bin = make([]byte, UDP_SIZE)
		n, l, err := this.conn.ReadFromUDP(bin)
		if err != nil {
			fmt.Println(err)
		}
		if this.remote == nil {
			this.remote = l
		}
		bin = bin[:n]
		el := NewProtocol().Decode(bin)

		switch el.Act {
		case PROTOCOL_HAND:
			once.Do(func() {
				this.able = true
				this.delay = int16(math.Ceil(float64(time.Now().Sub(el.time)) / float64(1000000)))
				go this.heartBeats(this.ctx)
			})
		case PROTOCOL_HEART_BEATS:
			//this.buffer.start = int(el.windowStart)
		case PROTOCOL_SERIAL:
			for _, v := range el.missList {
				go this.sendAgain(int(v))
			}
		case PROTOCOL_WINDOW:
			this.window.speedControll(float32(el.wr / 100))
			//case PROTOCOL_END:
			//this.Close()
		default:
			this.readChan <- el
			return
		}
	}
}

func (this *Tunnel) GetMissPackAndRequestAgain(miss []uint32) {
	if len(miss) <= 0 {
		return
	}
	p := NewProtocol()
	lenth := len(miss)
	p.Act = PROTOCOL_ACK
	p.id = this.getAckId()
	var buff = &bytes.Buffer{}
	for i := 0; i < 100 && i < lenth; i++ {
		j := miss[i]
		binary.Write(buff, binary.LittleEndian, &j)
	}
	p.data = buff.Bytes()
	this.conn.Write(p.EncodeMissPack())
}

func (this *Tunnel) Connect(addr string) error {
	udpAdr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}
	this.conn, err = net.DialUDP("udp", nil, udpAdr)
	if err != nil {
		return err
	}

	p := NewProtocol()
	p.Act = PROTOCOL_HAND
	p.time = time.Now()
	b := p.EncodeHandPack()
	time.Sleep(10 * time.Millisecond)
	this.conn.Write(b)
	this.conn.Write(b)
	this.conn.Write(b)
	go this.ReceiveMsgFromRemote()
	time.Sleep(100 * time.Millisecond)
	return nil
}

func (this *Tunnel) Listen(port int) (error, int) {
	err, p := this.listen(port)
	if err == nil {
		go this.Send()
		go this.send()
		go this.ReceiveMsgFromLocal()
		time.Sleep(50 * time.Millisecond)
	}
	return err, p
}

func (this *Tunnel) listen(port int) (error, int) {
	if port > 0 {
		udp, err := net.ResolveUDPAddr("udp", "0.0.0.0:"+strconv.Itoa(port))
		if err != nil {
			return err, 0
		}
		this.conn, err = net.ListenUDP("udp", udp)
		if err != nil {
			return err, 0
		}
		return nil, port
	}
	for port := 2000; port <= 9000; port++ {
		udp, err := net.ResolveUDPAddr("udp", "0.0.0.0:"+strconv.Itoa(port))
		if err != nil {
			continue
		}
		this.conn, err = net.ListenUDP("udp", udp)
		if err != nil {
			continue
		}
		fmt.Println("-listen")
		return nil, port
	}
	return LISTEN_FAIL, 0
}

func (this *Tunnel) speedControll(ctx context.Context) {
	var loadControll = func(i int) {
		el := NewProtocol()
		el.Act = PROTOCOL_WINDOW
		el.id = this.getAckId()
		switch {
		case i < 25:
			el.wr = 400
		case 25 <= i && i < 50:
			el.wr = 200
		case 50 <= i && i < 75:
			el.wr = 100
		case 75 <= i && i < 100:
			el.wr = 50
		case 100 <= i:
			el.wr = 0
		}
		this.conn.Write(el.EncodeWindowPack())
	}
	var congestion = func() {
		r := this.readBuffer.MissRate()
		el := NewProtocol()
		el.Act = PROTOCOL_WINDOW
		el.id = this.getAckId()
		switch {
		case r < 5:
			el.wr = 100
		case 5 <= r && r < 100:
			el.wr = 75
		case 100 <= r && r < 200:
			el.wr = 50
		case 500 < r:
			el.wr = 0
		}
		this.conn.Write(el.EncodeWindowPack())
	}
	for {
		time.Sleep(1 * time.Second)
		load := 100 * this.readBuffer.Len() / this.readBuffer.Size() //buffer 占用百分
		select {
		case <-ctx.Done():
			return
		default:
			loadControll(load) //缓存区控制
			congestion()       //拥塞控制
		}
	}
}

func (this *Tunnel) getAckId() uint16 {
	for i := uint16(1); i <= 255; i++ {
		if _, ok := this.ackList[i]; !ok {
			return i
		}
	}
	panic("ack id fail")
}

func (this *Tunnel) sendAgain(offset int) {
	el := this.writeBuffer.ReadOffset(offset)
	el.offset = uint32(offset)
	this.priorChan <- el.EncodeDataPack()
}

func (this *Tunnel) Read() *Protocol {
	return <-this.readChan
}
