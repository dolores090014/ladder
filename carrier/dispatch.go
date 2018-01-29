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
	"net/http"
	"net"
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
	if err != nil {
		panic(err)
	}
	if this.tunnel.Able() {
		panic("不通")
	}
}

func (this *dis) Listen(port int) {
	err, _ := this.tunnel.Listen(port)
	go this.tunnel.Send()
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

func (this *dis) Regester(c *carrier) {
	this.l.Lock()
	defer this.l.Unlock()
	this.m[c.requestId] = c
}

func (this *dis) Dispatch(el []*mUDP.Protocol) {
	if len(el) <= 0 {
		return
	}
	for _, v := range el {
		this.m[int(v.RequestId)].ch <- v
	}
}

func NewCarrier(tcp *net.TCPConn) *carrier {
	c := &carrier{
		conn: tcp,
	}
	c.requestId = Client.getID(c)
	if c.requestId == 0 {
		panic("get id fail")
	}
	go c.readRequestFromBrowser()
	go c.writeToLocalTCP()
	return c
}

func NewRequest(id int, bin []byte) {
	Remote.l.Lock()
	defer Remote.l.Unlock()
	if v, ok := Remote.m[id]; ok {
		v.conn.Write(bin)
	} else {
		//b1, _, _ := bufio.NewReader(bytes.NewReader(bin)).ReadLine()
		req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(bin)))
		if err != nil {
			fmt.Fprintln(os.Stderr, err, "h:", )
			return
		}
		port := 80
		if req.Method == "CONNECT" {
			port = 443
		}
		ips, err := net.LookupIP(strings.Split(req.Host, ":")[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, "get ip fail", err)
		}
		url := ips[0].String() + ":" + strconv.Itoa(port)
		adr2, err := net.ResolveTCPAddr("tcp", url)
		conn, err := net.DialTCP("tcp", nil, adr2)
		if err != nil {
			fmt.Fprintln(os.Stderr, "connect website fail", err)
			return
		}
		c := &carrier{
			conn: conn,
		}
		c.requestId = id
		Remote.Regester(c)

		go c.getDataFromWebSite()
		go c.readMsg()

		if req.Method == "CONNECT" {
			Remote.tunnel.SendMsgToLocal(uint16(c.requestId), []byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
		} else {
			c.conn.Write(bin)
		}
	}
}
