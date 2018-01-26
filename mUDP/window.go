package mUDP

import "time"

type window struct {
	value  int16
	status time.Timer //
	max    int16
	min    int16
}

func (this *window) window() int16 {
	return this.value
}

func (this *window) set(i int16) {
	this.value = i
}

func (this *window) speedControll(resp float32) {
	if this.value == 0 {
		return
	}
	switch {
	case resp > 1:
		this.min = this.value
		if this.min > 0 && this.max > 0 {
			this.value = this.min + int16(float32(this.max-this.min)*float32(resp))
		} else {
			this.value = int16(float32(this.value) * float32(resp))
		}
	case 0 < resp && resp <= 1:
		this.max = this.value
		if this.min > 0 && this.max > 0 {
			this.value = this.min + int16(float32(this.max-this.min)*float32(resp))
		} else {
			this.value = int16(float32(this.value) * float32(resp))
		}
	case resp == 0:
		this.max = this.value
		this.value = 0
		time.AfterFunc(1*time.Second, func() {
			this.value = this.min + int16(float32(this.max-this.min)*0.5)
		})
	}

}
