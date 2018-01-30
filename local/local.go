package main

import (
	"net"
	"time"
	"ladder/carrier"
	"ladder/common"
	"fmt"
	"runtime"
)

func forward(localTCP *net.TCPConn) {
	localTCP.SetDeadline(time.Unix(time.Now().Unix()+30, 0))
	carrier.NewCarrier(localTCP)
}

func tcpServer() {
	var tcpAddr *net.TCPAddr
	tcpAddr, _ = net.ResolveTCPAddr("tcp", "127.0.0.1:"+common.PROXY_PORT)
	tcpListener, _ := net.ListenTCP("tcp", tcpAddr)
	defer tcpListener.Close()
	for {
		tcpConn, err := tcpListener.AcceptTCP()
		if err != nil {
			continue
		}
		go forward(tcpConn)
	}
}

func main() {
	go func() {
		for{
			fmt.Println("gr:",runtime.NumGoroutine())
			time.Sleep(1*time.Second)
		}
	}()
	carrier.Client.Connect("127.0.0.1:2000")
	tcpServer()
}
