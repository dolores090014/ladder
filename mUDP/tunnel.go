package mUDP

import (
	"net"
	"fmt"
	"sync"
	"time"
	"context"
	"strconv"
	"math"
	"ladder/common"
)

type tunnel struct {
	ctx        context.Context
	able       bool
	delay      int16 //ms
	l          *sync.Mutex
	m          map[int]*Client
	r          map[int]*Carrier
	priorChan  chan []byte
	normalChan chan []byte
	//local      *net.UDPAddr
	remote     *net.UDPAddr
	conn       *net.UDPConn
}

var TunnelLocal = &tunnel{
	ctx: context.Background(),
	l:   &sync.Mutex{},
	m:   make(map[int]*Client),
}

var TunnelRemote = &tunnel{
	ctx:        context.Background(),
	priorChan:  make(chan []byte, 1024*512),
	normalChan: make(chan []byte, 1024*1024),
	l:          &sync.Mutex{},
	r:          make(map[int]*Carrier),
}

func (this *tunnel) Lock() {
	this.l.Lock()
}

func (this *tunnel) Unlock() {
	this.l.Unlock()
}

/*
local
 */
func (this *tunnel) Write(bin []byte) {
	this.conn.Write(bin)
}

/*
remote
 */
func (this *tunnel) PriorWrite(bin []byte, prior bool) {
	if prior {
		common.PushChan(bin, this.priorChan)
	} else {
		common.PushChan(bin, this.normalChan)
	}
}

func (this *tunnel) Able() bool {
	return this.able
}

func (this *tunnel) heartBeats(ctx context.Context) {
	for {
		time.Sleep(time.Duration(HEART_BEAT_INTERVAL) * time.Second)
		select {
		case <-ctx.Done():
			return
		default:
			p := NewProtocol()
			p.act = PROTOCOL_HEART_BEATS
			this.conn.Write(p.EncodeHeartBeatPack())
			//p.windowStart = uint32(this.buffer.start)
			//this.sendChan <- p.EncodeHeartBeatPack()
		}
	}
}

func (this *tunnel) Remove(e *Client) {
	this.l.Lock()
	defer this.l.Unlock()
	delete(this.m, e.id)
}

func (this *tunnel) getID(con *Client) int {
	max := 1<<16 - 1
	this.l.Lock()
	defer this.l.Unlock()
	for i := 1; i <= max; i++ {
		if _, ok := this.m[i]; !ok {
			this.m[i] = con
			return i
		}
	}
	return 0
}

/*
local receive
*/
func (this *tunnel) receive() {
	for {
		var bin = make([]byte, UDP_SIZE)
		n, err := this.conn.Read(bin)
		if err != nil {
			fmt.Println(err)
		}
		bin = bin[:n]
		el := NewProtocol().Decode(bin)
		this.l.Lock()
		defer func() {
			if recover()!=nil{
				fmt.Println(el.requestId)
			}
		}()
		this.m[int(el.requestId)].ch <- el
		this.l.Unlock()
	}
}

/*
remote receive
 */
func (this *tunnel) Receive() {
	once := &sync.Once{}
	for {
		var bin = make([]byte, UDP_SIZE)
		n,l ,err := this.conn.ReadFromUDP(bin)
		if err != nil {
			fmt.Println(err)
		}
		if this.remote == nil{
			this.remote = l
		}
		bin = bin[:n]
		el := NewProtocol().Decode(bin)
		if el.act == PROTOCOL_HAND {
			once.Do(func() {
				this.able = true
				this.delay = int16(math.Ceil(float64(time.Now().Sub(el.time)) / float64(1000000)))
				go this.heartBeats(this.ctx)
			})
		} else {
			this.l.Lock()
			if v, ok := this.r[int(el.requestId)]; ok {
				v.ch <- el
			}
			this.l.Unlock()
		}
	}
}

func (this *tunnel) Listen(port int) (error, int) {
	err, p := this.listen(port)
	if err == nil {
		go this.send()
		go this.Receive()
	}
	return err, p
}

func (this *tunnel) listen(port int) (error, int) {
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

func (this *tunnel) Connect(addr string) error {
	udpAdr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}
	this.conn, err = net.DialUDP("udp", nil, udpAdr)
	if err != nil {
		return err
	}

	p := NewProtocol()
	p.act = PROTOCOL_HAND
	p.time = time.Now()
	b := p.EncodeHandPack()

	go this.receive()

	time.Sleep(10 * time.Millisecond)
	this.Write(b)
	this.Write(b)
	this.Write(b)
	time.Sleep(100 * time.Millisecond)

	return nil
}

func (this *tunnel) send() {
	for {
		var bin []byte
		select {
		//case <-ctx.Done():
		//	close(this.priorChan)
		//	close(this.normalChan)
		//	return
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
		this.conn.WriteToUDP(bin,this.remote)
	}
}
