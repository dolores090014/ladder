package mUDP

import (
	"time"
	"encoding/binary"
	"bytes"
	"context"
)

type Client struct {
	id      int
	ctx     context.Context
	end     context.CancelFunc
	ackList map[uint16]*Protocol
	ch      chan *Protocol
	buffer  *RingBufferWithMiss
}

func NewClient() *Client {
	time.Sleep(100 * time.Millisecond)
	c := &Client{
		ch:     make(chan *Protocol, 32),
		buffer: NewWriteBuffer(1024 * 8),
	}
	c.ctx, c.end = context.WithCancel(context.Background())
	c.id = TunnelLocal.getID(c)
	go c.receive()
	//go c.speedControll(c.ctx)
	return c
}

func (this *Client) Id() int {
	return this.id
}

func (this *Client) Done() <-chan struct{} {
	return this.ctx.Done()
}

func (this *Client) Close() {
	el := NewProtocol()
	el.act = PROTOCOL_END
	TunnelLocal.Remove(this)
	TunnelLocal.Write(el.EncodeEndSign())
	TunnelLocal.Write(el.EncodeEndSign())
	TunnelLocal.Write(el.EncodeEndSign())
	this.end()
}

func (this *Client) Write(bin []byte) {
}

func (this *Client) Read(bin []byte) int {
	return this.buffer.Read(bin)
}

func (this *Client) speedControll(ctx context.Context) {
	var loadControll = func(i int) {
		el := NewProtocol()
		el.act = PROTOCOL_WINDOW
		el.id = this.getAckId()
		switch {
		case i < 25:
			el.wr = 400
		case 25 <= i && i < 50:
			el.wr = 200
		case 50 <= i && i < 75:
			el.wr = 100
		case 75 <= i && i < 100:
			el.wr = 50
		case 100 <= i:
			el.wr = 0
		}
		TunnelLocal.Write(el.EncodeWindowPack())
	}
	var congestion = func() {
		r := this.buffer.MissRate()
		el := NewProtocol()
		el.act = PROTOCOL_WINDOW
		el.id = this.getAckId()
		switch {
		case r < 5:
			el.wr = 100
		case 5 <= r && r < 100:
			el.wr = 75
		case 100 <= r && r < 200:
			el.wr = 50
		case 500 < r:
			el.wr = 0
		}
		TunnelLocal.Write(el.EncodeWindowPack())
	}
	for {
		time.Sleep(1 * time.Second)
		load := 100 * this.buffer.Len() / this.buffer.Size() //buffer 占用百分
		select {
		case <-ctx.Done():
			return
		default:
			loadControll(load) //缓存区控制
			congestion()       //拥塞控制
		}
	}
}

func (this *Client) GetMissPackAndRequestAgain(list []uint32) {
	p := NewProtocol()
	lenth := len(list)
	p.act = PROTOCOL_ACK
	p.id = this.getAckId()
	var buff = &bytes.Buffer{}
	for i := 0; i < 100 && i < lenth; i++ {
		j := list[i]
		binary.Write(buff, binary.LittleEndian, &j)
	}
	p.data = buff.Bytes()
	TunnelLocal.Write(p.EncodeMissPack())
}

func (this *Client) receive() {
	for {
		select {
		case <-this.ctx.Done():
			return
		case el := <-this.ch:
			switch el.act {
			case PROTOCOL_DATA:
				miss := this.buffer.Write(el, int(el.offset), int(el.windowStart))
				if len(miss) > 0 {
					this.GetMissPackAndRequestAgain(miss)
				}
			case PROTOCOL_ACK:
				if _, ok := this.ackList[el.id]; ok {
					delete(this.ackList, el.id)
				}
			}
		}
	}
}

func (this *Client) getAckId() uint16 {
	for i := uint16(1); i <= 255; i++ {
		if _, ok := this.ackList[i]; !ok {
			return i
		}
	}
	panic("ack id")
}
