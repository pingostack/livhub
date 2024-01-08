package auth

import (
	"context"

	"github.com/pingostack/neon/internal/core/middleware"
)

func Recovery() middleware.Middleware {
	return func(h middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			return h(ctx, req)
		}
	}
}
