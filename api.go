package talk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// Connect ...
func Connect(ctx context.Context, requestID string, flt Filter) <-chan Client {
	req := ConnectRequest{
		Req: NewReq(ctx, requestID),
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

// MsgJSON ...
type MsgJSON struct {
	Keys      map[interface{}]interface{} `json:"keys"`
	Content   interface{}                 `json:"content"`
	MsgStamp  int64                       `json:"msgstamp"`
	CreatedAt string                      `json:"createdAt"`
}

// SSEConnect ...
func SSEConnect(w http.ResponseWriter, r *http.Request, requestID string, flt Filter) <-chan struct{} {
	f, ok := w.(http.Flusher)
	if !ok {
		panic(errors.New("Flush() not supported"))
	}

	var cli Client
	select {
	case <-r.Context().Done():
	case cli = <-Connect(r.Context(), requestID, flt):
	}
	if cli == nil {
		return nil
	}

	w.Header().Set("Content-Type", "text/event-stream")

	go func() {
		for {
			select {
			case <-r.Context().Done():
				cli.Close()
				break
			case msg := <-cli.C():
				b, err := json.Marshal(MsgJSON{
					Keys:      msg.Keys,
					Content:   msg.Content,
					MsgStamp:  msg.MsgStamp,
					CreatedAt: time.Now().Format("2006-01-02 15:04:05"),
				})
				if err != nil {
					continue
				}
				formatStr := "event:message\ndata:%s\n\n"
				_, err = w.Write(bytes.NewBufferString(fmt.Sprintf(formatStr, string(b))).Bytes())
				if err != nil {
					continue
				}
				f.Flush()
			}
		}
	}()

	return cli.Done()
}

// WSConnect ...
func WSConnect(filters []Filter) <-chan struct{} {
	return nil
}

// Send ...
func Send(ctx context.Context, requestID string, keys []interface{}, content interface{}) interface{} {
	req := SendRequest{
		Req:     NewReq(ctx, requestID),
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
