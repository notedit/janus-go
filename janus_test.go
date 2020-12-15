package janus

import (
	"context"
	"testing"
)

func Test_Connect(t *testing.T) {

	client, err := Connect(context.TODO(), "ws://39.106.248.166:8188/")
	if err != nil {
		t.Fail()
		return
	}
	mess, err := client.Info(context.TODO())
	if err != nil {
		t.Fail()
		return
	}
	t.Log(mess)

	sess, err := client.Create(context.TODO())
	if err != nil {
		t.Fail()
		return
	}
	t.Log(sess)
	t.Log("connect")
}
