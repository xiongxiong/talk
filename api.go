package talk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// Connect ...
func Connect(ctx context.Context, flt Filter) Client {
	req := ConnectRequest{
		Req: NewReq(ctx),
		Flt: flt,
	}
	Request(req)
	res := <-req.ResCh()
	switch r := res.(type) {
	case Client:
		return r
	default:
		return nil
	}
}

// MsgJSON ...
type MsgJSON struct {
	Keys      map[string]interface{} `json:"keys"`
	Content   interface{}            `json:"content"`
	MsgStamp  int64                  `json:"msgstamp"`
	CreatedAt string                 `json:"createdAt"`
}

// SSEConnect ...
func SSEConnect(w http.ResponseWriter, r *http.Request, flt Filter) Client {
	var cli Client
	select {
	case <-r.Context().Done():
	default:
		cli = Connect(r.Context(), flt)
	}
	if cli == nil {
		return nil
	}

	w.Header().Set("Content-Type", "text/event-stream")

	w.Write(bytes.NewBufferString("event:.\ndata:.\n\n").Bytes())
	w.(http.Flusher).Flush()

	go func() {
		done := true
	DONE:
		for done {
			select {
			case <-r.Context().Done():
				done = false
				cli.Close()
				break DONE
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
				w.(http.Flusher).Flush()
			}
		}
	}()

	return cli
}

// WSConnect ...
func WSConnect(w http.ResponseWriter, r *http.Request, flt Filter) Client {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		panic(errors.New("connection cannot be established"))
	}

	var cli Client
	select {
	case <-r.Context().Done():
	default:
		cli = Connect(r.Context(), flt)
	}
	if cli == nil {
		return nil
	}

	w.Header().Set("Content-Type", "text/event-stream")

	go func() {
		done := true
	DONE:
		for done {
			select {
			case <-r.Context().Done():
				done = false
				cli.Close()
				break DONE
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
				if err = conn.WriteMessage(websocket.TextMessage, b); err != nil {
					continue
				}
			}
		}
	}()

	return cli
}

// Send ...
func Send(ctx context.Context, keys map[string]interface{}, content interface{}) interface{} {
	req := SendRequest{
		Req:     NewReq(ctx),
		Keys:    keys,
		Content: content,
	}
	Request(req)
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return <-req.ResCh()
	}
}
