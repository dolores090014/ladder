package main

import (
	"net"
	"net/http"
	"bytes"
	"crypto/tls"
	"fmt"
	"ladder/mUDP"
	"time"
	"strconv"
)

const REMOTE_ADDR = "127.0.0.1:1024"
const PROXY_PORT = "8086"

func forward(localTCP *net.TCPConn) {
	localTCP.SetDeadline(time.Unix(time.Now().Unix()+30, 0))
	client := mUDP.NewClient()
	/*
	response
	 */
	go func() {
		for {
			time.Sleep(40 * time.Millisecond)
			select {
			case <-client.Done():
				return
			default:
				var bin = make([]byte, 1024*8)
				n := client.Read(bin)
				bin = bin[:n]
				localTCP.Write(bin)
			}
		}
	}()

	/*
	读取浏览器请求
	 */
	go func() {
		for {
			bin := make([]byte, 1024*8)
			n, err := localTCP.Read(bin)
			bin = bin[:n]

			//http.ReadRequest(bufio.NewReader(bytes.NewReader(bin)))
			//if err == nil && strings.Contains(req.Host, "google") {
			//	client.Close()
			//	localTCP.Close()
			//	return
			//}

			httpsGet(REMOTE_ADDR+"?id="+strconv.Itoa(client.Id()), bin)
			if err != nil {
				fmt.Println("brower err:", err)
				//httpsGet(REMOTE_ADDR+"?port="+strconv.Itoa(int(port))+"&close=true", nil)
				client.Close()
				localTCP.Close()
				return
			}
		}
	}()
}

func httpsGet(url string, bin []byte) *http.Response {
	if bin == nil {
		bin = []byte("")
	}
	req, _ := http.NewRequest(
		"GET",
		"https://"+url,
		bytes.NewReader(bin))
	resp, err := (&http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}).Do(req)
	if err != nil {
		//todo
	}
	return resp
}

func tcpServer() {
	var tcpAddr *net.TCPAddr
	tcpAddr, _ = net.ResolveTCPAddr("tcp", "127.0.0.1:"+PROXY_PORT)
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
	err := mUDP.TunnelLocal.Connect("127.0.0.1:2000")
	fmt.Println(err)
	if mUDP.TunnelLocal.Able() {
		fmt.Println("不通")
	}
	tcpServer()
}
