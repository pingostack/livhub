package auth

import (
	"context"

	"github.com/pingostack/livhub/internal/core/middleware"
)

func Recovery() middleware.Middleware {
	return func(h middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req middleware.Request) (interface{}, error) {
			return h(ctx, req)
		}
	}
}
