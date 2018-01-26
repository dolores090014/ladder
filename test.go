package main

import (
	"net"
	"fmt"
	"time"
	"runtime"
	"bufio"
	"bytes"
	"strings"
	"net/http"
)

func main() {
	go func() {
		for {
			time.Sleep(1 * time.Second)
			fmt.Println(runtime.NumGoroutine())
		}
	}()
	var tcpAddr *net.TCPAddr
	tcpAddr, _ = net.ResolveTCPAddr("tcp", "0.0.0.0:8086")
	tcpListener, _ := net.ListenTCP("tcp", tcpAddr)
	for {
		tcpConn, err := tcpListener.AcceptTCP()
		if err != nil {
			continue
		}
		go func(c2 *net.TCPConn) {
			c2.SetKeepAlivePeriod(3 * time.Second)
			ta, _ := net.ResolveTCPAddr("tcp", "172.19.3.65:80")
			//ta, _ := net.ResolveTCPAddr("tcp", "47.95.163.175:443")
			c, _ := net.DialTCP("tcp", nil, ta)
			go func() {
				for i:=1;i<=3;i++{
					bin := make([]byte, 1024*512)
					n, _ := c2.Read(bin)
					bin = bin[:n]
					fmt.Println("i:",i,"l:",n)
				}
				for {
					bin := make([]byte, 1024*512)
					n, err := c2.Read(bin)
					bin = bin[:n]
					if err != nil {
						fmt.Println("close")
						c2.Close()
						c.Close()
						return
					}
					req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(bin)))
					if err != nil {
						fmt.Println(len(bin))
						l, _, _ := bufio.NewReader(bytes.NewReader(bin)).ReadLine()
						fmt.Println(string(l))
						fmt.Println()
						fmt.Println(string(bin))
						return
					} else {
						if strings.Contains(req.Host, "google") {
							c.Close()
							c2.Close()
							return
						}
					}
					if err == nil {
						if req.Method != "CONNECT" {
							c.Write(bin)
						}
						if req.Method == "CONNECT" {
							c2.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
						}
					} else {
						c.Write(bin)
					}
				}
			}()

			go func() {
				for {
					b := make([]byte, 1024*1024)
					n, err := c.Read(b)
					if err != nil {
						c.Close()
						return
					}
					b = b[:n]
					c2.Write(b)
				}
			}()

		}(tcpConn)
	}
}
