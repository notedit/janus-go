package janus

import (
	"context"
	"testing"
)

func Test_Connect(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // cancel when we are finished consuming integers

	


	
	client, err := Connect(ctx,"ws://localhost:8188/")
	if err != nil {
		t.Fail()
		return
	}

	go client.Receiver(ctx)
	go client.Ping(ctx)



	mess, err := client.Info(ctx)
	if err != nil {
		t.Fail()
		return
	}
	t.Log(mess)

	sess, err := client.Create(ctx)
	if err != nil {
		t.Fail()
		return
	}
	t.Log(sess)
	t.Log("connect")
}
