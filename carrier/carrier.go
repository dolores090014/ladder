package carrier

import (
	"net"
	"ladder/mUDP"
	"context"
	"strconv"
	"fmt"
	"ladder/common"
	"os"
)

type carrier struct {
	conn      *net.TCPConn
	ctx       context.Context
	cancel    context.CancelFunc
	requestId uint16
	ch        chan *mUDP.Protocol
}

/*
receive and write localTCP
 */
func (this *carrier) writeToLocalTCP() {
	defer func() {
		fmt.Println("die2->id",this.requestId)
	}()
	for {
		select {
		case <-this.ctx.Done():
			return
		default:
			el,ok := <-this.ch
			if !ok{
				return
			}
			_,err:=this.conn.Write(el.Data())
			if err!=nil{
				return
			}
		}
	}
}

func (this *carrier) readRequestFromBrowser() {
	defer func() {
		fmt.Println("die->id",this.requestId)
	}()
	for {
		bin := make([]byte, 1024*8)
		n, err := this.conn.Read(bin)
		if err != nil {
			fmt.Println("brower err:", err)
			this.CloseLocal()
			return
		}
		if n == 0 {
			continue
		}
		bin = bin[:n]
		common.HttpsGet(common.REMOTE_ADDR+"?id="+strconv.Itoa(int(this.requestId)), bin)
	}
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
				this.CloseRemote()
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

func (this *carrier) CloseLocal() {
	defer func() { recover() }()
	el := mUDP.NewProtocol()
	el.SetAct(mUDP.PROTOCOL_END)
	el.RequestId = uint16(this.requestId)
	Client.remove(this)
	Client.tunnel.Write(el.EncodeEndSign())
	Client.tunnel.Write(el.EncodeEndSign())
	Client.tunnel.Write(el.EncodeEndSign())
	this.cancel()
	close(this.ch)
	this.conn.Close()
}
func (this *carrier) CloseRemote() {
	defer func() { recover() }()
	this.cancel()
	this.conn.Close()
}

func (this *carrier) readMsg() {
	for {
		select {
		case <-this.ctx.Done():
			return
		default:
			el,ok := <-this.ch
			if !ok{
				return
			}
			switch el.Act {
			case mUDP.PROTOCOL_END:
				this.cancel()
			}
		}
	}
}
