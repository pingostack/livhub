package pms

import (
	"context"
	"net/http"
	"strings"

	"github.com/bluenviron/gortsplib/v4/pkg/sdp"
	"github.com/gin-gonic/gin"
	"github.com/gogf/gf/util/guid"
	"github.com/pingostack/neon/internal/core/router"
	"github.com/pingostack/neon/internal/httpserv"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type SignalServer struct {
	*httpserv.SignalServer
	ctx    context.Context
	logger *logrus.Entry
}

func NewSignalServer(ctx context.Context, httpParams httpserv.HttpParams, logger *logrus.Entry) *SignalServer {
	return &SignalServer{
		SignalServer: httpserv.NewSignalServer(ctx, httpParams, logger),
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
	// parse sdp
	sd := sdp.SessionDescription{}
	if err := sd.Unmarshal([]byte(req.Data.SDP)); err != nil {
		return err
	}

	hasAudio := false
	hasVideo := false
	hasDataChannel := false
	for _, mediaDesc := range sd.MediaDescriptions {
		if mediaDesc.MediaName.Media == "audio" {
			hasAudio = true
		} else if mediaDesc.MediaName.Media == "video" {
			hasVideo = true
		} else if mediaDesc.MediaName.Media == "application" {
			hasDataChannel = true
		}
	}

	if !hasAudio && !hasVideo && !hasDataChannel {
		return errors.New("no audio/video/datachannel")
	}

	// create session
	peerID := req.Session
	if peerID == "" {
		peerID = guid.S()
	}
	pm := router.PeerMeta{
		RemoteAddr:     gc.Request.RemoteAddr,
		LocalAddr:      gc.Request.Host,
		PeerID:         peerID,
		RouterID:       req.Stream,
		Domain:         gc.Request.Host,
		URI:            gc.Request.URL.Path,
		Producer:       true,
		HasAudio:       hasAudio,
		HasVideo:       hasVideo,
		HasDataChannel: hasDataChannel,
	}

	session := NewSession(ss.ctx, pm, ss.logger)

	err := session.Join()
	if err != nil {
		return errors.Wrap(err, "join failed")
	}

	return nil
}

func (ss *SignalServer) play(req Request, gc *gin.Context) error {
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
