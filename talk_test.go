package talk_test

import (
	"sync"
	"talk"
	"talk/req"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBroadcast(t *testing.T) {
	t.Log("broadcast should work")

	talk.Serve()

	s := [2]struct{}{}
	wg := sync.WaitGroup{}
	content := "hello"

	m := make(map[uuid.UUID]*talk.MsgOut)
	for range s {
		wg.Add(1)
		go func() {
			defer wg.Done()
			uid := uuid.New()
			req := talk.ConnectRequest{
				Req:  req.NewReq(nil),
				UUID: uid,
			}
			talk.Request(req)
			cli, err := req.Res()
			if err != nil {
				t.Errorf("connect failure -- %s", err.Error())
			}
			msg := <-cli.C
			m[uid] = msg
		}()
	}

	time.Sleep(time.Second)
	bst := talk.BroadcastRequest{
		Req: req.NewReq(nil),
		MsgIn: talk.MsgIn{
			Content: content,
		},
	}
	talk.Request(bst)
	res := <-bst.ResCh()
	switch res.(type) {
	case nil, error:
		t.Error("send broadcast request failure")
	}

	wg.Wait()

	if len(m) != len(s) {
		t.Error("cannot broadcast to all")
	}

	for _, v := range m {
		if v.Content != content {
			t.Error("msg content is wrong")
		}
		if v.MsgStamp != 1 {
			t.Error("msg stamp is wrong")
		}
	}
}

func TestUnicast(t *testing.T) {

}
