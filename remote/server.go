package main

import (
	"net/http"
	"io/ioutil"
	"fmt"
	"time"
	"runtime"
	"strconv"
	"ladder/carrier"
	"ladder/common"
)

func server() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		bin, _ := ioutil.ReadAll(r.Body)
		//rl,_,err :=bufio.NewReader(bytes.NewReader(bin)).ReadLine()
		//method :=
		//if err!=nil{
		//	fmt.Fprintln(os.Stderr,err)
		//}
		//request, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(bin)))
		//if err != nil {
		//	fmt.Fprintln(os.Stderr,err)
		//	return
		//}
		q := r.URL.Query()
		id, _ := strconv.Atoi(q.Get("id"))
		if q.Get("close") != "" {
			//todo mUDP.CloseRequest(id)
		} else {
			carrier.NewRequest(id, bin)
		}
		return
	})
	server := &http.Server{Addr: "0.0.0.0:" + common.REMOTE_PORT, Handler: nil}
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
	carrier.Remote.Listen(common.UDP_PORT)
	server()
}
