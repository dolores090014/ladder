package main

import (
	"net/http"
	"bufio"
	"io/ioutil"
	"bytes"
	"fmt"
	"time"
	"ladder/mUDP"
	"runtime"
	"strconv"
)

const REMOTE_PORT = "1024"
const UDP_PORT = 2000

func server() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		bin, _ := ioutil.ReadAll(r.Body)
		request, _ := http.ReadRequest(bufio.NewReader(bytes.NewReader(bin)))
		if request == nil {
			return
		}
		q := r.URL.Query()
		id, _ := strconv.Atoi(q.Get("id"))
		if q.Get("close") != "" {
			mUDP.CloseRequest(id)
		} else {
			mUDP.NewRequest(id, bin)
		}
		return
	})
	server := &http.Server{Addr: "0.0.0.0:" + REMOTE_PORT, Handler: nil}
	server.SetKeepAlivesEnabled(false)
	err := server.ListenAndServeTLS("./server.crt", "./server.key")
	if err != nil {
		fmt.Println(err)
	}
}

func main() {
	go func() {
		for {
			time.Sleep(1 * time.Second)
			fmt.Println("cr:", runtime.NumGoroutine())
		}
	}()
	err, _ := mUDP.TunnelRemote.Listen(UDP_PORT)
	if err != nil {
		panic(err)
	}
	server()
}
