package logging

import (
	"context"
	"fmt"

	"github.com/pingostack/livhub/internal/core/middleware"
	"github.com/sirupsen/logrus"
)

func Recovery(logger *logrus.Entry) middleware.Middleware {
	return func(h middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req middleware.Request) (interface{}, error) {
			reply, err := h(ctx, req)
			logger.Infof("request: %s, reply: %s, err: %v", toString(req), toString(reply), err)
			return reply, err
		}
	}
}

func toString(x interface{}) string {
	if x == nil {
		return "none"
	}

	if stringer, ok := x.(fmt.Stringer); ok {
		return stringer.String()
	}
	return fmt.Sprintf("%+v", x)
}
