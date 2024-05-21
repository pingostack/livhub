package whip

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/pingostack/neon/internal/httpserv"
	"github.com/sirupsen/logrus"
)

type SignalServer struct {
	ss     *httpserv.SignalServer
	ctx    context.Context
	logger *logrus.Entry
}

func NewSignalServer(ctx context.Context, httpParams httpserv.HttpParams, logger *logrus.Entry) *SignalServer {
	return &SignalServer{
		ss:     httpserv.NewSignalServer(ctx, httpParams, logger),
		ctx:    ctx,
		logger: logger,
	}
}

func (ss *SignalServer) Start() error {
	return ss.ss.Start(ss.handleRequest)
}

func (ss *SignalServer) Close() error {
	return ss.ss.Close()
}

func (ss *SignalServer) handleRequest(gc *gin.Context) {
	ss.logger.Infof("request: %s %s", gc.Request.Method, gc.Request.URL.String())
	gc.JSON(200, gin.H{
		"message": "pong",
	})
}
