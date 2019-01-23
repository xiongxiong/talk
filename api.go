package talk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"talk/req"
	"time"
)

// Connect ...
func Connect(ctx context.Context, flt Filter) <-chan Client {
	req := ConnectRequest{
		Req: req.NewReq(ctx),
		Flt: flt,
	}
	Request(req)
	c := make(chan Client)
	go func() {
		res := <-req.ResCh()
		switch r := res.(type) {
		case Client:
			c <- r
		default:
			c <- nil
		}
	}()
	return c
}

type MsgJson struct {
	Keys      map[interface{}]interface{} `json:"keys"`
	Content   interface{}                 `json:"content"`
	MsgStamp  int64                       `json:"msgstamp"`
	CreatedAt string                      `json:"createdAt"`
}

// SSEConnect ...
func SSEConnect(w http.ResponseWriter, r *http.Request, flt Filter) {
	f, ok := w.(http.Flusher)
	if !ok {
		panic(errors.New("Flush() not supported"))
	}

	var cli Client
	select {
	case <-r.Context().Done():
	case cli = <-Connect(r.Context(), flt):
	}
	if cli == nil {
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")

	formatStr := "event:message\ndata:%s\n\n"
	for {
		select {
		case <-r.Context().Done():
			cli.Close()
			break
		case msg := <-cli.C():
			b, err := json.Marshal(MsgJson{
				Keys:      msg.Keys,
				Content:   msg.Content,
				MsgStamp:  msg.MsgStamp,
				CreatedAt: time.Now().Format("2006-01-02 15:04:05"),
			})
			if err != nil {
				continue
			}
			_, err = w.Write(bytes.NewBufferString(fmt.Sprintf(formatStr, string(b))).Bytes())
			if err != nil {
				continue
			}
			f.Flush()
		}
	}
}

// WSConnect ...
func WSConnect(filters []Filter) <-chan Client {
	c := make(chan Client)
	return c
}

// Send ...
func Send(keys []interface{}, content interface{}) interface{} {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := SendRequest{
		Req:     req.NewReq(ctx),
		Content: content,
	}
	Request(req)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case res := <-req.ResCh():
		return res
	}
}
