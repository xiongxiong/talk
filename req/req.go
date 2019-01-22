package req

import (
	"context"
)

// Req ...
type Req interface {
	Ctx() context.Context
	ResCh() chan interface{}
}

// Base ...
type Base struct {
	ctx   context.Context
	resCh chan interface{}
}

// Ctx ...
func (r *Base) Ctx() context.Context {
	return r.ctx
}

// ResCh ...
func (r *Base) ResCh() chan interface{} {
	return r.resCh
}

// NewReq ...
func NewReq(ctx context.Context) *Base {
	if ctx == nil {
		ctx = context.TODO()
	}
	return &Base{
		ctx:   ctx,
		resCh: make(chan interface{}, 1),
	}
}
