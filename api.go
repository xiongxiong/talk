package talk

import (
	"context"
	"errors"
	"talk/req"
	"time"
)

// Connect ...
func Connect(filters []Filter) (Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := ConnectRequest{
		Req:     req.NewReq(ctx),
		Filters: filters,
	}
	Request(req)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-req.ResCh():
		switch r := res.(type) {
		case Client:
			return r, nil
		default:
			return nil, errors.New("connect failure")
		}
	}
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
