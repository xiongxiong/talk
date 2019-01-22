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
		done:  make(chan struct{}),
		reqCh: make(chan req.Req, 10),
		clients: clientMap{
			clients: make(map[*ConnectRequest]*client),
		},
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
	done    chan struct{}
	reqCh   chan req.Req
	clients clientMap
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
				t.clients = clientMap{
					clients: make(map[*ConnectRequest]*client),
				}
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
	return len(t.clients.clients)
}

// Msg ...
type Msg struct {
	Content  interface{}
	MsgStamp int64
}

// Filter ...
type Filter func(keys []interface{}) bool

// Done ...
type Done interface {
	Done() <-chan struct{}
	Close()
}

// Conn ...
type Conn interface {
	C() <-chan *Msg
}

type conn struct {
	c        chan *Msg
	msgStamp int64
}

func (c conn) C() <-chan *Msg {
	return c.c
}

func (c conn) MsgStamp() int64 {
	atomic.AddInt64(&c.msgStamp, 1)
	return c.msgStamp
}

// Client ...
type Client interface {
	Done
	GetConn(filter *Filter) Conn
}

type client struct {
	connMap map[*Filter]conn
	done    chan struct{}
}

func (c *client) Done() <-chan struct{} {
	return c.done
}

func (c *client) Close() {
	c.done <- struct{}{}
}

func (c *client) GetConn(filter *Filter) Conn {
	return c.connMap[filter]
}

type clientMap struct {
	sync.RWMutex
	clients map[*ConnectRequest]*client
}

// ConnectRequest ...
type ConnectRequest struct {
	req.Req
	Filters []Filter
}

func connect(t *Talk, req *ConnectRequest) {
	select {
	case <-req.Ctx().Done():
		req.ResCh() <- req.Ctx().Err()
	default:
		connM := make(map[*Filter]conn)
		for _, f := range req.Filters {
			connM[&f] = conn{
				c: make(chan *Msg, 1),
			}
		}
		cli := client{
			connMap: connM,
			done:    make(chan struct{}),
		}

		t.clients.Lock()
		t.clients.clients[req] = &cli
		t.clients.Unlock()

		go func() {
			<-cli.done
			t.clients.Lock()
			delete(t.clients.clients, req)
			t.clients.Unlock()
		}()

		req.ResCh() <- &cli
	}
}

// SendRequest ...
type SendRequest struct {
	req.Req
	Content interface{}
	Keys    []interface{}
}

func send(t *Talk, req *SendRequest) {
	select {
	case <-req.Ctx().Done():
		req.ResCh() <- req.Ctx().Err()
	default:
		t.clients.RLock()
		defer t.clients.RUnlock()
		for _, cli := range t.clients.clients {
			for f, c := range cli.connMap {
				if (*f)(req.Keys) {
					msg := Msg{
						Content:  req.Content,
						MsgStamp: c.MsgStamp(),
					}
					select {
					case c.c <- &msg:
					default:
					}
				}
			}
		}
		req.ResCh() <- "success"
	}
}
