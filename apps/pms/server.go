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
	interRtc "github.com/pingostack/neon/internal/rtc"
	"github.com/pingostack/neon/pkg/deliver/rtc"
	"github.com/pingostack/neon/pkg/rtclib/sdpassistor"
	"github.com/pion/webrtc/v4"
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

	src, err := rtc.NewFrameSource(ss.ctx, interRtc.StreamFactory(), false, settings().KeyFrameIntervalSecond*time.Second, logger)
	if err != nil {
		logger.WithError(err).Error("failed to create frame source")
		return errors.Wrap(err, "failed to create frame source")
	}

	err = src.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  req.Data.SDP,
	})
	if err != nil {
		logger.WithError(err).Error("failed to set remote description")
		return errors.Wrap(err, "failed to set remote description")
	}

	_, err = src.CreateAnswer(nil)
	if err != nil {
		logger.WithError(err).Error("failed to create answer")
		return errors.Wrap(err, "failed to create answer")
	}

	err = src.Start()
	if err != nil {
		logger.WithError(err).Error("failed to start frame source")
		return errors.Wrap(err, "failed to start frame source")
	}

	logger.WithField("metadata", src.Metadata().String()).Debug("frame source metadata")

	// create session
	session := NewSession(ss.ctx, router.PeerParams{
		RemoteAddr:     gc.Request.RemoteAddr,
		LocalAddr:      gc.Request.Host,
		PeerID:         peerID,
		RouterID:       req.Stream,
		Domain:         gc.Request.Host,
		URI:            gc.Request.URL.Path,
		Producer:       true,
		HasAudio:       src.Metadata().HasAudio(),
		HasVideo:       src.Metadata().HasVideo(),
		HasDataChannel: src.Metadata().HasData(),
	}, logger)

	err = session.BindFrameSource(src)
	if err != nil {
		logger.WithError(err).Error("failed to bind frame source")
		return errors.Wrap(err, "failed to bind frame source")
	}

	err = session.Join()
	if err != nil {
		logger.WithError(err).Error("join failed")
		return errors.Wrap(err, "join failed")
	}

	lsdp, err := src.GatheringCompleteLocalSdp(context.Background())
	if err != nil {
		logger.WithError(err).Error("failed to get completed sdp")
		return errors.Wrap(err, "failed to get completed sdp")
	}

	resp := Response{
		Version: req.Version,
		Method:  req.Method,
		Err:     0,
		ErrMsg:  "",
		Session: session.ID(),
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
	// create session
	peerID := req.Session
	if peerID == "" {
		peerID = guid.S()
	}

	logger := ss.logger.WithFields(logrus.Fields{
		"session": peerID,
		"stream":  req.Stream,
	})

	hasAudio, hasVideo, hasData, err := sdpassistor.GetPayloadStatus(req.Data.SDP, webrtc.SDPTypeOffer)
	if err != nil {
		logger.WithError(err).Error("failed to get payload status")
		return errors.Wrap(err, "failed to get payload status")
	}

	session := NewSession(ss.ctx, router.PeerParams{
		RemoteAddr:     gc.Request.RemoteAddr,
		LocalAddr:      gc.Request.Host,
		PeerID:         peerID,
		RouterID:       req.Stream,
		Domain:         gc.Request.Host,
		URI:            gc.Request.URL.Path,
		Producer:       false,
		HasAudio:       hasAudio,
		HasVideo:       hasVideo,
		HasDataChannel: hasData,
	}, logger)

	dest, err := rtc.NewFrameDestination(ss.ctx, interRtc.StreamFactory(),
		false, logger)
	if err != nil {
		logger.WithError(err).Error("failed to create frame destination")
		return errors.Wrap(err, "failed create frame destination")
	}

	err = dest.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  req.Data.SDP,
	})
	if err != nil {
		logger.WithError(err).Error("failed to set remote description")
		return errors.Wrap(err, "failed to set remote description")
	}

	err = session.BindFrameDestination(dest)
	if err != nil {
		logger.WithError(err).Error("failed to bind frame destination")
		return errors.Wrap(err, "failed to bind frame source")
	}

	err = session.Join()
	if err != nil {
		if errors.Is(err, router.ErrPaddingDestination) {
			timeout := settings().JoinTimeoutSecond
			select {
			case <-ss.ctx.Done():
				return errors.Wrap(err, "context done")
			case err = <-dest.SourceCompletePromise():
				if err != nil {
					logger.WithError(err).Error("join failed")
					return errors.Wrap(err, "join failed")
				} else {
					logger.Info("join success")
				}
			case <-time.After(timeout * time.Second):
				logger.WithField("timeout", timeout).Error("join timeout")
				return errors.New("join timeout")
			}
		} else {
			logger.WithError(err).Error("join failed")
			return errors.Wrap(err, "join failed")
		}
	}

	_, err = dest.CreateAnswer(nil)
	if err != nil {
		logger.WithError(err).Error("failed to create answer")
		return errors.Wrap(err, "failed to create answer")
	}

	lsdp, err := dest.GatheringCompleteLocalSdp(context.Background())
	if err != nil {
		logger.WithError(err).Error("failed to get completed sdp")
		return errors.Wrap(err, "failed to get completed sdp")
	}

	err = dest.Start()
	if err != nil {
		logger.WithError(err).Error("failed to start frame destination")
		return errors.Wrap(err, "failed to start frame destination")
	}

	resp := Response{
		Version: req.Version,
		Method:  req.Method,
		Err:     0,
		ErrMsg:  "",
		Session: session.ID(),
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
