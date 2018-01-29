package common

import (
	"errors"
	"bytes"
	"net/http"
	"crypto/tls"
)

const REMOTE_PORT = "1024"
const UDP_PORT = 2000
const REMOTE_ADDR = "127.0.0.1:1024"
const PROXY_PORT = "8086"

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

func HttpsGet(url string, bin []byte) *http.Response {
	if bin == nil {
		bin = []byte("")
	}
	req, _ := http.NewRequest(
		"GET",
		"https://"+url,
		bytes.NewReader(bin))
	resp, err := (&http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}).Do(req)
	if err != nil {
		//todo
	}
	return resp
}
