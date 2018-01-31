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
)

type dis struct {
	l      *sync.Mutex
	m      map[int]*carrier
	tunnel *mUDP.Tunnel
}

var Client *dis
var Remote *dis

func init() {
	Client = &dis{
		l:      new(sync.Mutex),
		m:      make(map[int]*carrier),
		tunnel: mUDP.NewTunnel(true),
	}
	Remote = &dis{
		l:      new(sync.Mutex),
		m:      make(map[int]*carrier),
		tunnel: mUDP.NewTunnel(false),
	}
}

func (this *dis) Connect(addr string) {
	err := this.tunnel.Connect(addr)
	go this.Dispatch()
	if err != nil {
		panic(err)
	}
	if this.tunnel.Able() {
		panic("不通")
	}
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
	delete(this.m, int(e.requestId))
}

func (this *dis) getID(con *carrier) int {
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

func (this *dis) Register(c *carrier) {
	this.l.Lock()
	defer this.l.Unlock()
	this.m[c.requestId] = c
}

func (this *dis) Dispatch() {
	for {
		el := this.tunnel.Read()
		this.l.Lock()
		this.m[int(el.RequestId)].ch <- el
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

func NewCarrier2(id int, tcp *net.TCPConn) *carrier {
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
	if v, ok := Remote.m[id]; ok {
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
		c := NewCarrier2(id, conn)
		if method == "CONNECT" {
			Remote.tunnel.SendMsgToLocal(uint16(c.requestId), []byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
		} else {
			c.conn.Write(bin)
		}
	}
}
