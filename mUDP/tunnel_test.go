package mUDP

import (
	"testing"
	"time"
	"fmt"
)

func TestTunnel_Listen(t *testing.T) {
	r := NewTunnel(false)
	r.Listen(2000)
	c := NewTunnel(true)
	c.Connect("127.0.0.1:2000")
	//c.conn.SetReadBuffer(1024*1024)
	//c.conn.SetWriteBuffer(1024*1024)
	//r.conn.SetWriteBuffer(1024*1024)
	//r.conn.SetReadBuffer(1024*1024)
	go func() {
		time.Sleep(1 * time.Second)
		for i := 1; i <= 9999; i++ {
			r.SendMsgToLocal(uint16(i), []byte("-"))
		}
	}()

	go func() {
		k := 0
		for {
			c.Read()
			k += 1
			fmt.Println(k)
		}
	}()

	time.Sleep(1000 * time.Second)
}

func TestTunnel_Connect(t *testing.T) {

}
