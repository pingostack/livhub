package pms

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gogf/gf/util/guid"
	"github.com/pingostack/neon/internal/core/router"
	"github.com/pingostack/neon/internal/httpserv"
	inter_rtc "github.com/pingostack/neon/internal/rtc"
	"github.com/pingostack/neon/pkg/deliver/rtc"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type SignalServer struct {
	*httpserv.SignalServer
	ctx    context.Context
	logger *logrus.Entry
}

func NewSignalServer(ctx context.Context, logger *logrus.Entry) *SignalServer {
	return &SignalServer{
		SignalServer: httpserv.NewSignalServer(ctx, settings().HttpParams, logger),
		ctx:          ctx,
		logger:       logger,
	}
}

func (ss *SignalServer) Start() error {
	return ss.SignalServer.Start(ss.handleRequest)
}

func (ss *SignalServer) Close() error {
	return ss.SignalServer.Close()
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

	if err := ss.handleRequestInternal(req, gc); err != nil {
		gc.JSON(http.StatusInternalServerError, gin.H{
			"message": "internal server error",
		})
		return
	}

}

func (ss *SignalServer) handleRequestInternal(req Request, gc *gin.Context) error {
	switch req.Method {
	case "stream.publish":
		return ss.publish(req, gc)
	case "stream.play":
		return ss.play(req, gc)
	case "stream.close":
		return ss.close(req, gc)
	case "stream.mute":
		return ss.mute(req, gc)
	case "stream.max_bitrate":
		return ss.maxBitrate(req, gc)
	}

	return nil
}

func (ss *SignalServer) publish(req Request, gc *gin.Context) error {
	peerID := req.Session
	if peerID == "" {
		peerID = guid.S()
	}
	logger := ss.logger.WithFields(logrus.Fields{
		"session": peerID,
		"stream":  req.Stream,
	})

	s := rtc.NewServSession(ss.ctx, inter_rtc.StreamFactory(), router.PeerParams{
		RemoteAddr: gc.Request.RemoteAddr,
		LocalAddr:  gc.Request.Host,
		PeerID:     peerID,
		RouterID:   req.Stream,
		Domain:     gc.Request.Host,
		URI:        gc.Request.URL.Path,
		Producer:   true,
	}, logger)

	lsdp, err := s.Publish(settings().KeyFrameIntervalSecond*time.Second, req.Data.SDP)
	if err != nil {
		logger.WithError(err).Error("failed to publish")
		return errors.Wrap(err, "failed to publish")
	}

	resp := Response{
		Version: req.Version,
		Method:  req.Method,
		Err:     0,
		ErrMsg:  "",
		Session: peerID,
		Data: struct {
			SDP string `json:"sdp"`
		}{
			SDP: lsdp.SDP,
		},
	}

	logger.WithField("answer", lsdp.SDP).Debug("resp answer")
	gc.JSON(http.StatusOK, resp)

	return nil
}

func (ss *SignalServer) play(req Request, gc *gin.Context) error {
	peerID := req.Session
	if peerID == "" {
		peerID = guid.S()
	}
	logger := ss.logger.WithFields(logrus.Fields{
		"session": peerID,
		"stream":  req.Stream,
	})
	domain := gc.Request.Host
	sp := strings.Split(domain, ":")
	if len(sp) > 0 {
		domain = sp[0]
	}

	s := rtc.NewServSession(ss.ctx, inter_rtc.StreamFactory(), router.PeerParams{
		RemoteAddr: gc.Request.RemoteAddr,
		LocalAddr:  gc.Request.Host,
		PeerID:     peerID,
		RouterID:   req.Stream,
		Domain:     domain,
		URI:        gc.Request.URL.Path,
		Producer:   true,
	}, logger)

	lsdp, err := s.Subscribe(req.Data.SDP, settings().JoinTimeoutSecond*time.Second)
	if err != nil {
		logger.WithError(err).Error("failed to subscribe")
		return errors.Wrap(err, "failed to subscribe")
	}

	resp := Response{
		Version: req.Version,
		Method:  req.Method,
		Err:     0,
		ErrMsg:  "",
		Session: peerID,
		Data: struct {
			SDP string `json:"sdp"`
		}{
			SDP: lsdp.SDP,
		},
	}

	logger.WithField("answer", lsdp.SDP).Debug("resp answer")
	gc.JSON(http.StatusOK, resp)

	return nil
}

func (ss *SignalServer) close(req Request, gc *gin.Context) error {
	return nil
}

func (ss *SignalServer) mute(req Request, gc *gin.Context) error {
	return nil
}

func (ss *SignalServer) maxBitrate(req Request, gc *gin.Context) error {
	return nil
}
