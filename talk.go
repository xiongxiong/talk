package talk

import (
	"errors"
	"sync"
	"talk/req"

	"github.com/google/uuid"
)

var (
	talk = NewTalk()
)

// NewTalk ...
func NewTalk() *Talk {
	return &Talk{
		done:      make(chan signal),
		reqCh:     make(chan req.Req, 10),
		clientMap: make(map[uuid.UUID]*client),
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

// Talk ...
type Talk struct {
	sync.RWMutex
	done      chan signal
	reqCh     chan req.Req
	clientMap map[uuid.UUID]*client
	msgStamp  int64
}

// Request ...
func (t *Talk) Request(req req.Req) {
	switch r := req.(type) {
	case ConnectRequest,
		BroadcastRequest,
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
				case BroadcastRequest:
					go broadcast(t, &r)
				case SendRequest:
					go unicast(t, &r)
				}
			case <-t.done:
				break
			}
		}
	}()
}

func (t *Talk) Stop() {
	// TODO
	close(t.done)
}

func (t *Talk) connCount() int {
	t.RLock()
	defer t.RUnlock()

	return len(t.clientMap)
}

type client struct {
	c         chan *MsgOut
	done      chan signal
	filterMap FilterMap
}

// Client ...
type Client struct {
	C    <-chan *MsgOut
	Done chan<- signal
}

// FilterMap ...
type FilterMap struct {
	sync.RWMutex
	Map map[Filter]int64 // filter -- msgstamp
}

// Filter ...
type Filter interface {
	// Filt return true if permit to send
	Filt(keywords []string) bool
}

type signal struct{}

// MsgIn ...
type MsgIn struct {
	Content string
}

// MsgOut ...
type MsgOut struct {
	MsgIn
	MsgStamp int64
}

// ConnectRequest ...
type ConnectRequest struct {
	req.Req
	UUID    uuid.UUID
	Filters []Filter
}

// Res ...
func (r *ConnectRequest) Res() (*Client, error) {
	res := <-r.ResCh()
	if cli, ok := res.(*Client); ok {
		return cli, nil
	}
	return nil, errors.New("connect failure")
}

func connect(t *Talk, req *ConnectRequest) {
	t.RLock()
	cli := t.clientMap[req.UUID]
	if cli != nil {
		close(cli.done)
	}
	t.RUnlock()

	cli = &client{
		c:    make(chan *MsgOut, 10),
		done: make(chan signal),
		filterMap: FilterMap{
			Map: map[Filter]int64{},
		},
	}
	cli.filterMap.Lock()
	for _, f := range req.Filters {
		cli.filterMap.Map[f] = 0
	}
	cli.filterMap.Unlock()

	t.Lock()
	t.clientMap[req.UUID] = cli
	t.Unlock()

	go waitToClean(t, req.UUID)

	req.ResCh() <- &Client{
		C:    cli.c,
		Done: cli.done,
	}
}

func waitToClean(t *Talk, uid uuid.UUID) {
	t.RLock()
	cli := t.clientMap[uid]
	t.RUnlock()

	if cli != nil {
		_, open := <-cli.done
		if open {
			close(cli.done)
		}
		close(cli.c)
	}
	t.Lock()
	delete(t.clientMap, uid)
	t.Unlock()
}

// BroadcastRequest ...
type BroadcastRequest struct {
	req.Req
	MsgIn
}

func broadcast(t *Talk, req *BroadcastRequest) {
	t.msgStamp++
	msg := MsgOut{
		MsgIn:    req.MsgIn,
		MsgStamp: t.msgStamp,
	}
	for _, cli := range t.clientMap {
		if cli != nil {
			cli.c <- &msg
		}
	}
	req.ResCh() <- "success"
}

// SendRequest ...
type SendRequest struct {
	req.Req
	MsgIn
	Keywords []string
}

func unicast(t *Talk, req *SendRequest) {
	for _, cli := range t.clientMap {
		if cli != nil {
			cli.filterMap.Lock()
			for f := range cli.filterMap.Map {
				if f.Filt(req.Keywords) {
					cli.filterMap.Map[f]++
					cli.c <- &MsgOut{
						MsgIn:    req.MsgIn,
						MsgStamp: cli.filterMap.Map[f],
					}
					break
				}
			}
			cli.filterMap.Unlock()
		}
	}
}
