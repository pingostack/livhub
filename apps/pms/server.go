package pms

import (
	"context"
	"net/http"
	"strings"

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

	if !strings.EqualFold(gc.Request.Method, http.MethodPost) {
		gc.JSON(http.StatusMethodNotAllowed, gin.H{
			"message": "method not allowed",
		})
		return
	}

	req := Request{}
	if err := gc.ShouldBindJSON(&req); err != nil {
		gc.JSON(http.StatusBadRequest, gin.H{
			"message": "bad request",
		})
		return
	}

	if err := ss.handleRequestInternal(req); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{
			"message": "internal server error",
		})
		return
	}
}

func (ss *SignalServer) handleRequestInternal(req Request) error {
	switch req.Method {
	case "stream.publish":
		return ss.publish(req)
	case "stream.play":
		return ss.play(req)
	case "stream.close":
		return ss.close(req)
	case "stream.mute":
		return ss.mute(req)
	case "stream.max_bitrate":
		return ss.maxBitrate(req)
	}

	return nil
}

func (ss *SignalServer) publish(req Request) error {

	return nil
}

func (ss *SignalServer) play(req Request) error {
	return nil
}

func (ss *SignalServer) close(req Request) error {
	return nil
}

func (ss *SignalServer) mute(req Request) error {
	return nil
}

func (ss *SignalServer) maxBitrate(req Request) error {
	return nil
}
