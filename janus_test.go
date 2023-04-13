package janus

import (
	"net/http"
	"testing"
)

func Test_Connect(t *testing.T) {

	reqHeader := http.Header{"User-Agent": []string{"janus-test"}}
	client, err := Connect("ws://39.106.248.166:8188/", reqHeader)
	if err != nil {
		t.Fail()
		return
	}
	mess, err := client.Info()
	if err != nil {
		t.Fail()
		return
	}
	t.Log(mess)

	sess, err := client.Create()
	if err != nil {
		t.Fail()
		return
	}
	t.Log(sess)
	t.Log("connect")
}
