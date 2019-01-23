package talk

import (
	"errors"
	"sync"
	"sync/atomic"
	"talk/req"
)

var (
	talk = NewTalk()
)

// NewTalk ...
func NewTalk() *Talk {
	return &Talk{
		done:    make(chan struct{}),
		reqCh:   make(chan req.Req, 10),
		clients: make(map[*Filter]*client),
	}
}

// Request ...
func Request(req req.Req) {
	talk.Request(req)
}

// Start ...
func Start() {
	talk.Start()
}

// Stop ...
func Stop() {
	talk.Stop()
}

// ClientCount ...
func ClientCount() int {
	return talk.ClientCount()
}

// Talk ...
type Talk struct {
	sync.RWMutex
	done    chan struct{}
	reqCh   chan req.Req
	clients map[*Filter]*client
	db      DB
}

// Request ...
func (t *Talk) Request(req req.Req) {
	switch r := req.(type) {
	case ConnectRequest,
		SendRequest:
		t.reqCh <- req
	default:
		r.ResCh() <- errors.New("request not supported")
	}
}

// Start ...
func (t *Talk) Start() {
	go func() {
		for {
			select {
			case req := <-t.reqCh:
				switch r := req.(type) {
				case ConnectRequest:
					go connect(t, &r)
				case SendRequest:
					go send(t, &r)
				}
			case <-t.done:
				t.Lock()
				t.clients = make(map[*Filter]*client)
				t.Unlock()
				break
			}
		}
	}()
}

// Stop ...
func (t *Talk) Stop() {
	select {
	case t.done <- struct{}{}:
	default:
	}
}

// ClientCount ...
func (t *Talk) ClientCount() int {
	return len(t.clients)
}

// DB ...
type DB interface {
	Save(msg DBMsg) error
}

// DBMsg ...
type DBMsg interface {
	MsgKeys() map[interface{}]interface{}
	MsgContent() interface{}
}

// Msg ...
type Msg struct {
	Keys     map[interface{}]interface{}
	Content  interface{}
	MsgStamp int64
}

// Filter ...
type Filter func(keys map[interface{}]interface{}) bool

// Closer ...
type Closer interface {
	Close()
}

// Client ...
type Client interface {
	Closer
	C() <-chan *Msg
}

type client struct {
	c        chan *Msg
	done     chan struct{}
	msgStamp int64
}

func (c *client) Close() {
	c.done <- struct{}{}
}

func (c *client) C() <-chan *Msg {
	return c.c
}

func (c *client) MsgStamp() int64 {
	atomic.AddInt64(&c.msgStamp, 1)
	return c.msgStamp
}

// ConnectRequest ...
type ConnectRequest struct {
	req.Req
	Flt Filter
}

func connect(t *Talk, req *ConnectRequest) {
	select {
	case <-req.Ctx().Done():
		req.ResCh() <- req.Ctx().Err()
	default:
		cli := client{
			c:    make(chan *Msg),
			done: make(chan struct{}),
		}

		t.Lock()
		t.clients[&req.Flt] = &cli
		t.Unlock()

		go func() {
			<-cli.done
			t.Lock()
			delete(t.clients, &req.Flt)
			t.Unlock()
		}()

		req.ResCh() <- &cli
	}
}

// SendRequest ...
type SendRequest struct {
	req.Req
	Keys    map[interface{}]interface{}
	Content interface{}
}

// MsgKeys ...
func (r *SendRequest) MsgKeys() map[interface{}]interface{} {
	return r.Keys
}

// MsgContent ...
func (r *SendRequest) MsgContent() interface{} {
	return r.Content
}

func send(t *Talk, req *SendRequest) {
	select {
	case <-req.Ctx().Done():
		req.ResCh() <- req.Ctx().Err()
	default:
		if t.db != nil {
			err := t.db.Save(req)
			if err != nil {
				req.ResCh() <- err
				return
			}
		}
		t.RLock()
		defer t.RUnlock()
		for flt, cli := range t.clients {
			if (*flt)(req.Keys) {
				msg := Msg{
					Keys:     req.Keys,
					Content:  req.Content,
					MsgStamp: cli.MsgStamp(),
				}
				select {
				case cli.c <- &msg:
				default:
				}
			}
		}
		req.ResCh() <- "success"
	}
}
