package common

import (
	"errors"
)

var CLOSED_CHAN = errors.New("closed chan")

func PushChan(bin []byte, c chan []byte) (err error) {
	defer func() {
		if err := recover(); err != nil {
			err = CLOSED_CHAN
		}
	}()
	c <- bin
	return nil
}
