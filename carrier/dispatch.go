package carrier

import (
	"sync"
	"ladder/mUDP"
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"
	"strconv"
	"net"
	"context"
	"net/http"
	"time"
)

type dis struct {
	l      *sync.Mutex
	m      map[uint16]*carrier
	poor   []uint16
	tunnel *mUDP.Tunnel
}

var Client *dis
var Remote *dis

func init() {
	Client = &dis{
		l:      new(sync.Mutex),
		m:      make(map[uint16]*carrier),
		poor:   make([]uint16, 1),
		tunnel: mUDP.NewTunnel(true),
	}
	Remote = &dis{
		l:      new(sync.Mutex),
		m:      make(map[uint16]*carrier),
		tunnel: mUDP.NewTunnel(false),
	}
}

func (this *dis) Connect(addr string) {
	err := this.tunnel.Connect(addr)
	this.createID()
	go this.Dispatch()
	if err != nil {
		panic(err)
	}
	if this.tunnel.Able() {
		panic("不通")
	}
}

func (this *dis) createID() {
	this.l.Lock()
	defer this.l.Unlock()

	if len(this.poor) >= 20 {
		return
	}
	var MAX = 99
	p := make([]uint16, 100)
	t := time.Now()
	m := (t.Minute()%10)*10000 + t.Second()*100
	for i := 0; i <= MAX; i++ {
		p[i] = uint16(m + i)
	}
	this.poor = p
}

func (this *dis) Listen(port int) {
	err, _ := this.tunnel.Listen(port)
	go this.Dispatch()
	if err != nil {
		panic(err)
	}
}

func (this *dis) remove(e *carrier) {
	this.l.Lock()
	defer this.l.Unlock()
	delete(this.m, uint16(e.requestId))
}

func (this *dis) getID(con *carrier) uint16 {
	this.l.Lock()
	id := this.poor[0]
	this.poor = this.poor[1:]
	this.m[id] = con
	this.l.Unlock()
	this.createID()
	return id
}

func (this *dis) Register(c *carrier) {
	this.l.Lock()
	defer this.l.Unlock()
	this.m[c.requestId] = c
}

func (this *dis) Dispatch() {
	//i:=0
	for {
		el := this.tunnel.Read()
		//i++
		//fmt.Println("-->",i)
		//continue
		this.l.Lock()
		if v,ok:=this.m[el.RequestId];ok{
			v.ch<-el
		}
		this.l.Unlock()
	}
}

func NewCarrier(tcp *net.TCPConn) *carrier {
	ctx, cancel := context.WithCancel(context.Background())
	c := &carrier{
		conn:   tcp,
		ctx:    ctx,
		ch:     make(chan *mUDP.Protocol, 32),
		cancel: cancel,
	}
	c.requestId = Client.getID(c)
	if c.requestId == 0 {
		panic("get id fail")
	}
	go c.readRequestFromBrowser()
	go c.writeToLocalTCP()
	return c
}

func NewCarrier2(id uint16, tcp *net.TCPConn) *carrier {
	ctx, cancel := context.WithCancel(context.Background())
	c := &carrier{
		conn:   tcp,
		ctx:    ctx,
		ch:     make(chan *mUDP.Protocol, 32),
		cancel: cancel,
	}
	c.requestId = id
	Remote.Register(c)
	go c.getDataFromWebSite()
	go c.readMsg()
	return c
}

func NewRequest(id int, bin []byte) {
	Remote.l.Lock()
	if v, ok := Remote.m[uint16(id)]; ok {
		Remote.l.Unlock()
		v.conn.Write(bin)
	} else {
		Remote.l.Unlock()
		//b1, _, _ := bufio.NewReader(bytes.NewReader(bin)).ReadLine()

		/***********************
		reader := bufio.NewReader(bytes.NewReader(bin))
		l1, _, _ := reader.ReadLine()
		method := strings.Split(string(l1), " ")[0]
		l2, _, _ := reader.ReadLine()
		host := strings.Split(string(l2), ":")[1]
		fmt.Println("l2:", string(l2))
		host = strings.Replace(host, " ", "", -1)
		/***********************/
		req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(bin)))
		if err != nil {
			fmt.Fprintln(os.Stderr, "connect website fail", err)
			return
		}
		method := req.Method
		host := req.Host
		fmt.Println(id, "-", req.URL)
		if method == "" || host == "" {
			fmt.Fprintln(os.Stderr, "analysis request fail")
			fmt.Fprintln(os.Stderr, string(bin))
			return
		}
		port := 80
		if method == "CONNECT" {
			port = 443
		}
		ips, err := net.LookupIP(strings.Split(host, ":")[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, "get ip fail", err)
		}
		url := ips[0].String() + ":" + strconv.Itoa(port)
		adr2, err := net.ResolveTCPAddr("tcp", url)
		if err != nil {
			fmt.Fprintln(os.Stderr, "connect website fail", err)
		}
		conn, err := net.DialTCP("tcp", nil, adr2)
		if err != nil {
			fmt.Fprintln(os.Stderr, "connect website fail", err)
			return
		}
		c := NewCarrier2(uint16(id), conn)
		if method == "CONNECT" {
			Remote.tunnel.SendMsgToLocal(uint16(c.requestId), []byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
		} else {
			c.conn.Write(bin)
		}
	}
}
