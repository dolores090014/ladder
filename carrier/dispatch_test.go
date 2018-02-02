package carrier

import (
	"testing"
	"time"
)

func TestDis_Listen(t *testing.T) {
	r := Remote
	r.Listen(2000)
	c := Client
	c.Connect("127.0.0.1:2000")
	//c..conn.SetReadBuffer(1024*1024)
	//c.conn.SetWriteBuffer(1024*1024)
	//r.conn.SetWriteBuffer(1024*1024)
	//r.conn.SetReadBuffer(1024*1024)
	go func() {
		time.Sleep(1 * time.Second)
		for i := 1; i <= 5000; i++ {
			b:=make([]byte,600)
			r.tunnel.SendMsgToLocal(uint16(i), b)
		}
	}()

	//go func() {
	//	k := 0
	//	for {
	//		c.tunnel.Read()
	//		k += 1
	//		fmt.Println(k)
	//	}
	//}()

	time.Sleep(1000 * time.Second)
}
