package mUDP

import (
	"sync"
)

/*
可跳跃的环形缓冲
 */

type buf struct {
	l     *sync.Mutex
	start int
	end   int
	bin   []byte
}

func (this *buf) Read(bin []byte) int {
	this.l.Lock()
	defer this.l.Unlock()
	if this.start >= this.end {
		return 0
	}
	var lenth = len(bin)
	var n = 0
	var size = len(this.bin)
	if this.end-this.start < lenth {
		n = this.end - this.start
	} else {
		n = lenth
	}
	if this.start%size+n <= size {
		copy(bin, this.bin[this.start%size:this.start%size+n])
	} else {
		s := this.bin[this.start%size:]
		copy(bin, s)
		copy(bin[len(s):], this.bin[:n-len(s)])
	}
	this.start += n
	return n
}

func (this *buf) Write(bin []byte) {
	this.l.Lock()
	defer this.l.Unlock()
	size := len(this.bin)
	lenth := len(bin)
	if size-(this.end-this.start) < lenth {
		return
	}
	//for i:=1;i<=lenth;i++{
	//	this.c<-1
	//}
	//填满右端，在下一个周期填充左端
	copy(this.bin[this.end%size:], bin)
	if size-this.end%size < lenth {
		copy(this.bin[:], bin[size-this.end%size:])
	}
	this.end += lenth
}

/*
可读区长度
 */
func (this *buf) readAble() int {
	return this.end - this.start
}

type RingBufferWithMiss struct {
	buf      *buf
	start    int
	end      int
	size     int
	element  []*Protocol
	missRate int
	//hit      []int8
}

func NewWriteBuffer(size int) *RingBufferWithMiss {
	return &RingBufferWithMiss{
		size: size,
		buf: &buf{
			l: &sync.Mutex{},
			//c:   make(chan int8, size*128),
			bin: make([]byte, size*128),
		},
		element: make([]*Protocol, size),
		//hit:     make([]int8, size),
	}
}

func (this *RingBufferWithMiss) List() []*Protocol {
	return this.element
}

func (this *RingBufferWithMiss) Len() int {
	return this.end - this.start
}

func (this *RingBufferWithMiss) ReadAble() int {
	return this.buf.readAble()
}

//func (this *RingBufferWithMiss) Hit() []int8 {
//	return this.hit
//}

func (this *RingBufferWithMiss) Size() int {
	return this.size
}

//func (this *RingBufferWithMiss) AllDone(start int, end int) bool {
//	for i := start; i <= end; i++ {
//		if this.hit[i] != 1 {
//			return false
//		}
//	}
//	return true
//}

func (this *RingBufferWithMiss) Bin() []*Protocol {
	return this.element
}

//func (this *RingBufferWithMiss) Read(bin []byte) int {
//	return this.buf.Read(bin)
//}

//func (this *RingBufferWithMiss) Read(bin []byte) int {
//	return this.buf.Read(bin)
//}

func (this *RingBufferWithMiss) Read() *Protocol {
	var el *Protocol
	if this.element[this.start+1%this.size]!=nil{
		this.start += 1
		el = this.element[this.start%this.size]
		this.element[this.start%this.size] = nil
	}
	return el
}

func (this *RingBufferWithMiss) Write(data *Protocol, offset int) {
	if this.size<=this.end-this.start{
		panic("full buffer")
	}
	if this.end <= offset {
		this.end = offset + 1
	}
	this.element[offset%this.size] = data
}

func (this *RingBufferWithMiss) Miss(wsd int) []uint32 {
	var miss = make([]uint32, 0)
	var mp = true
	if this.start >= wsd { //起点小于窗口左端
		mp = false
	}
	for i := this.start; ; i++ { //更新hit
		if i >= this.end {
			break
		}
		if mp {
			if this.element[i%this.size] == nil {
				miss = append(miss, uint32(i))
			}
		} else {
			break
		}
	}
	if len(miss) > 0 {
		this.missRate = len(miss)
	}
	return miss
}

func (this *RingBufferWithMiss) MissRate() int {
	if this.end > this.start {
		return this.missRate * 10000 / (this.end - this.start)
	}
	return 0
}

/*
********************************************  borderline   ***********************************************
 */

type SimpleRingBuffer struct {
	start     int
	end       int
	able      chan int8
	autoStart bool
	count     int
	size      int
	element   []*Protocol
}

func NewSimpleRingBuffer(size int) *SimpleRingBuffer {
	return &SimpleRingBuffer{
		size:    size,
		able:    make(chan int8, size),
		element: make([]*Protocol, size),
	}
}

func (this *SimpleRingBuffer) List() []*Protocol {
	return this.element
}

func (this *SimpleRingBuffer) Len() int {
	return this.count
}

func (this *SimpleRingBuffer) AutoUpdateStartPoint(b bool) {
	this.autoStart = b
}

func (this *SimpleRingBuffer) UpdateStartPoint(i int) {
	if this.autoStart || i == 0 {
		return
	}
	this.start += i
}

func (this *SimpleRingBuffer) Size() int {
	return this.size
}

func (this *SimpleRingBuffer) ReadOffset(offset int) *Protocol {
	return this.element[offset/this.size]
}

func (this *SimpleRingBuffer) Read(num int) ([]*Protocol, int, int) {
	o := this.start
	if this.start >= this.end {
		return nil, 0, 0
	}
	var r = make([]*Protocol, num)
	var n = 0
	if this.end-this.start < num {
		n = this.end - this.start
	} else {
		n = num
	}
	for i := 1; ; i++ {
		<-this.able
		this.count -= 1
		if i == n {
			break
		}
	}
	this.count -= n
	if this.start%this.size+n <= this.size {
		copy(r, this.element[this.start%this.size:this.start%this.size+n])
	} else {
		s := this.element[this.start%this.size:]
		copy(r, this.element[this.start%this.size:])
		copy(r[len(s):], this.element[:n-len(s)])
	}
	this.start += n
	return r, n, o
}

func (this *SimpleRingBuffer) Write(data *Protocol) {
	this.able <- 1
	this.count += 1
	this.element[this.end%this.size] = data
	this.end += 1
}
