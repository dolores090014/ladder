package mUDP

import (
	"errors"
	"encoding/binary"
	"time"
	"fmt"
)

var LISTEN_FAIL = errors.New("listen fail")

const UDP_SIZE = 512
const HEART_BEAT_INTERVAL = 1
const (
	PROTOCOL_HAND        = 1 //带时间戳测延迟
	PROTOCOL_HEART_BEATS = 2
	PROTOCOL_END         = 3 //结束信号
	PROTOCOL_ACK         = 4 //用于部分信息的确认
	PROTOCOL_SERIAL      = 5 //请求重新传输部分包
	PROTOCOL_WINDOW      = 6 //调整发送窗口
	PROTOCOL_DATA        = 7
)

type Protocol struct {
	act         int8      //-1
	id          uint16    //--需要ack的包的编号
	wr          uint16    //---窗口大小倍率
	windowStart uint32    //----
	windowSize  uint32    //----
	offset      uint32    //----
	time        time.Time //-----
	data        []byte    //------
	missList    []uint32  //-------
	requestId   uint16
	bin         []byte
}

func (this *Protocol) Bin() []byte {
	return this.bin
}

func NewProtocol() *Protocol {
	return &Protocol{
		data: make([]byte, 501),
		bin:  make([]byte, 512),
	}
}

/*
	数据包
	act -> data
	window start
	offset
	binary
 */
func (this *Protocol) EncodeDataPack() []byte {
	defer func() {
		if recover()!=nil{
			fmt.Println(this)
		}
	}()
	this.bin[0] = byte(this.act)
	binary.LittleEndian.PutUint32(this.bin[1:5], this.windowStart)
	binary.LittleEndian.PutUint32(this.bin[5:9], this.offset)
	binary.LittleEndian.PutUint16(this.bin[9:11], this.requestId)
	copy(this.bin[11:], this.data)
	this.bin = this.bin[:11+len(this.data)]
	return this.bin
}

//func (this *Protocol) SetAct(act int)  {
//	this.act = PROTOCOL_HEART_BEATS
//}

func (this *Protocol) EncodeHeartBeatPack() []byte {
	this.bin[0] = byte(this.act)
	binary.LittleEndian.PutUint32(this.bin[1:5], this.windowStart)
	this.bin = this.bin[:5]
	return this.bin
}

func (this *Protocol) EncodeWindowPack() []byte {
	this.bin[0] = byte(this.act)
	binary.LittleEndian.PutUint16(this.bin[1:3], this.wr)
	this.bin = this.bin[:3]
	return this.bin
}

func (this *Protocol) EncodeHandPack() []byte {
	u := this.time.UnixNano()
	this.bin[0] = byte(this.act)
	binary.LittleEndian.PutUint64(this.bin[1:9], uint64(u))
	this.bin = this.bin[:9]
	return this.bin
}

func (this *Protocol) EncodeMissPack() []byte {
	this.bin[0] = byte(this.act)
	copy(this.bin[1:], this.data)
	this.bin = this.bin[:1+len(this.data)]
	return this.bin
}

func (this *Protocol) EncodeEndSign() []byte {
	this.bin[0] = byte(this.act)
	binary.LittleEndian.PutUint16(this.bin[1:3], this.requestId)
	this.bin = this.bin[:3]
	return this.bin
}

func (this *Protocol) Decode(bin []byte) *Protocol {
	this.decodeAct(bin)
	switch this.act {
	case PROTOCOL_HAND:
		this.decodeHandPack(bin)
	case PROTOCOL_HEART_BEATS:
		this.decodeHeartBeatPack(bin)
	case PROTOCOL_END:
		return this
	case PROTOCOL_ACK:
	case PROTOCOL_SERIAL:
	case PROTOCOL_DATA:
		this.decodeDataPack(bin)
	}
	return this
}

func (this *Protocol) decodeHandPack(bin []byte) *Protocol {
	this.time = time.Unix(0, int64(binary.LittleEndian.Uint64(bin[1:9])))
	return this
}

func (this *Protocol) decodeHeartBeatPack(bin []byte) *Protocol {
	this.windowStart = binary.LittleEndian.Uint32(bin[1:5])
	return this
}

func (this *Protocol) decodeDataPack(bin []byte) *Protocol {
	this.windowStart = binary.LittleEndian.Uint32(bin[1:5])
	this.offset = binary.LittleEndian.Uint32(bin[5:9])
	this.requestId = binary.LittleEndian.Uint16(bin[9:11])
	this.data = bin[11:]
	return this
}

func (this *Protocol) decodeEndSign(bin []byte) *Protocol {
	this.requestId = binary.LittleEndian.Uint16(bin[1:3])
	return this
}

func (this *Protocol) decodeWindowPack(bin []byte) *Protocol {
	this.wr = binary.LittleEndian.Uint16(bin[1:3])
	return this
}

func (this *Protocol) decodeAct(bin []byte) *Protocol {
	this.act = int8(bin[0])
	return this
}

func (this *Protocol) decodeMissPack(bin []byte) *Protocol {
	this.bin = bin[1:]
	this.missList = make([]uint32, 0)
	for i := 0; i < len(bin)/8; i++ {
		this.missList = append(this.missList, binary.LittleEndian.Uint32(bin[i*8:(i+1)*8]))
	}
	return this
}

func BenchmarkMain() {
	el := NewProtocol()
	el.act = PROTOCOL_DATA
	el.windowStart = 233
	el.windowSize = 555
	el.offset = 666
	el.bin = []byte{233}
	el.EncodeDataPack()
}

func BenchmarkMain2() {
	el := &Protocol{}
	el.act = PROTOCOL_DATA
	el.windowStart = 233
	el.windowSize = 555
	el.offset = 666
	el.bin = []byte{233}
	var bin = make([]byte, 512)
	bin[0] = byte(el.act)
	binary.LittleEndian.PutUint32(bin[1:5], el.windowStart)
	binary.LittleEndian.PutUint32(bin[5:9], el.windowSize)
	binary.LittleEndian.PutUint32(bin[9:13], el.offset)
	copy(bin[13:], el.bin)
}
