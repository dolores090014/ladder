package main

import (
	"net"
	"fmt"
	"bufio"
	"bytes"
	"net/http"
)

func main() {
	var tcpAddr *net.TCPAddr
	tcpAddr, _ = net.ResolveTCPAddr("tcp", "0.0.0.0:8086")
	tcpListener, _ := net.ListenTCP("tcp", tcpAddr)
	for {
		tcpConn, err := tcpListener.AcceptTCP()
		if err != nil {
			continue
		}
		go func(c2 *net.TCPConn) {
			ta, _ := net.ResolveTCPAddr("tcp", "172.19.3.65:80")
			//ta, _ := net.ResolveTCPAddr("tcp", "183.230.40.96:443")
			//http.ListenAndServe()
			c, _ := net.DialTCP("tcp", nil, ta)
			go func() {
				for {
					bin := make([]byte, 1024*1024)
					n, err := c2.Read(bin)
					bin = bin[:n]
					if err != nil {
						fmt.Println(err)
						c2.Close()
						c.Close()
						return
					}
					req, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(bin)))
					fmt.Println(req)
					if err == nil {
						if req.Method != "CONNECT" {
							fmt.Println(string(bin))
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
