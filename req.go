package talk

import (
	"context"
)

// Req ...
type Req interface {
	Ctx() context.Context
	ID() string
	ResCh() chan interface{}
}

// Base ...
type Base struct {
	id    string
	ctx   context.Context
	resCh chan interface{}
}

// Ctx ...
func (r *Base) Ctx() context.Context {
	return r.ctx
}

// ID ...
func (r *Base) ID() string {
	return r.id
}

// ResCh ...
func (r *Base) ResCh() chan interface{} {
	return r.resCh
}

// NewReq ...
func NewReq(ctx context.Context, id string) *Base {
	if ctx == nil {
		ctx = context.TODO()
	}
	return &Base{
		ctx:   ctx,
		id:    id,
		resCh: make(chan interface{}, 1),
	}
}
