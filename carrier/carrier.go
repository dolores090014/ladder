package carrier

import (
	"net"
	"ladder/mUDP"
	"context"
	"time"
	"strconv"
	"fmt"
	"ladder/common"
	"os"
)

type carrier struct {
	conn      *net.TCPConn
	ctx       context.Context
	cancel    context.CancelFunc
	requestId int
	ch        chan *mUDP.Protocol
}

/*
receive and write localTCP
 */
func (this *carrier) writeToLocalTCP() {
	for {
		time.Sleep(40 * time.Millisecond)
		select {
		case <-this.ctx.Done():
			return
		default:
			el := <-this.ch
			this.conn.Write(el.Data())
		}
	}
}

func (this *carrier) readRequestFromBrowser() {
	for {
		bin := make([]byte, 1024*8)
		n, err := this.conn.Read(bin)
		if n == 0 {
			continue
		}
		bin = bin[:n]
		common.HttpsGet(common.REMOTE_ADDR+"?id="+strconv.Itoa(this.requestId), bin)
		if err != nil {
			fmt.Println("brower err:", err)
			this.Close()
			return
		}
	}
}

func (this *carrier) end() {
	this.cancel()
	this.conn.Close()
}

func (this *carrier) getDataFromWebSite() {
	for {
		select {
		case <-this.ctx.Done():
		default:
			bin := make([]byte, 1024)
			n, err := this.conn.Read(bin)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				this.Close()
				return
			}
			err = Remote.tunnel.SendMsgToLocal(uint16(this.requestId), bin[:n])
			if err != nil {
				fmt.Println("upd channel close")
				return
			}
		}
		//this.chanForDataBack <- bin[:n]
	}
}

func (this *carrier) Close() {
	el := mUDP.NewProtocol()
	el.SetAct(mUDP.PROTOCOL_END)
	el.RequestId = uint16(this.requestId)
	Client.remove(this)
	Client.tunnel.Write(el.EncodeEndSign())
	Client.tunnel.Write(el.EncodeEndSign())
	Client.tunnel.Write(el.EncodeEndSign())
	this.end()
}

func (this *carrier) readMsg() {
	for {
		select {
		case <-this.ctx.Done():
			return
		default:
			el := <-this.ch
			switch el.Act {
			case mUDP.PROTOCOL_END:
				this.cancel()
			}
		}
	}
}
