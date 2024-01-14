package middleware

import "context"

type Request struct {
	Operation string      `json:"operation"`
	Params    interface{} `json:"params"`
}

type Middleware func(Handler) Handler

type Handler func(ctx context.Context, req Request) (interface{}, error)

func Chain(m ...Middleware) Middleware {
	return func(next Handler) Handler {
		for i := len(m) - 1; i >= 0; i-- {
			next = m[i](next)
		}

		return next
	}
}
