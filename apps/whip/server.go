package whip

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gogf/gf/util/guid"
	"github.com/google/uuid"
	"github.com/let-light/gomodule"
	feature_rtc "github.com/pingostack/neon/features/rtc"
	"github.com/pingostack/neon/internal/core/router"
	"github.com/pingostack/neon/internal/httpserv"
	inter_rtc "github.com/pingostack/neon/internal/rtc"
	"github.com/pingostack/neon/pkg/deliver/rtc"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type SignalServer struct {
	ss         *httpserv.SignalServer
	ctx        context.Context
	logger     *logrus.Entry
	httpParams httpserv.HttpParams
	rtc        feature_rtc.Feature
}

func NewSignalServer(ctx context.Context, httpParams httpserv.HttpParams, logger *logrus.Entry) *SignalServer {
	ss := &SignalServer{
		ss:         httpserv.NewSignalServer(ctx, httpParams, logger),
		ctx:        ctx,
		logger:     logger,
		httpParams: httpParams,
	}

	gomodule.RequireFeatures(func(rtc feature_rtc.Feature) {
		ss.rtc = rtc
	})

	return ss
}

const (
	PathVarType   = "type"
	PathVarApp    = "app"
	PathVarStream = "stream"
	PathVarSecret = "secret"
)

func (ss *SignalServer) Start() error {
	// 配置 CORS 中间件
	corsConfig := cors.Config{
		AllowOrigins:     []string{"*"}, // 允许所有来源
		AllowMethods:     []string{"OPTIONS", "GET", "POST", "PATCH", "DELETE"},
		AllowHeaders:     []string{"Content-Type", "Authorization", "If-Match"},
		ExposeHeaders:    []string{"Link"},
		AllowCredentials: true,
	}

	ss.ss.DefaultRouter().Use(cors.New(corsConfig))
	ss.ss.DefaultRouter().RedirectTrailingSlash = false

	ss.ss.DefaultRouter().Use(func(gc *gin.Context) {
		gc.Writer.Header().Set("Access-Control-Allow-Origin", strings.Join(ss.httpParams.AllowOrigin, " ,"))
		gc.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	})

	ss.ss.DefaultRouter().Any("/:type/:app/:stream", ss.handleRequest)
	ss.ss.DefaultRouter().Any("/:type/:app/:stream/:secret", ss.handleRequest)

	return ss.ss.Start(func(gc *gin.Context) {
		if gc.Request.Method == http.MethodOptions && gc.Request.Header.Get("Access-Control-Request-Method") != "" {
			gc.Writer.Header().Set("Access-Control-Allow-Methods", "OPTIONS, GET, POST, PATCH, DELETE")
			gc.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, If-Match")
			gc.Writer.WriteHeader(http.StatusNoContent)
			return
		}
	})
}

func (ss *SignalServer) Close() error {
	return ss.ss.Close()
}

func (ss *SignalServer) getLinkHeader() []string {
	iceServers := ss.rtc.GetSettings().DefaultSettings.ICEServers
	link := []string{}
	for _, iceServer := range iceServers {
		l, err := iceServer.ToWhipLinkHeader()
		if err != nil {
			ss.logger.Errorf("ToWhipLinkHeader error: %v", err)
			continue
		}

		link = append(link, l...)
	}

	return link
}

func (ss *SignalServer) handleOptions(gc *gin.Context) {
	gc.Writer.Header().Set("Access-Control-Allow-Methods", "OPTIONS, GET, POST, PATCH, DELETE")
	gc.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, If-Match")
	gc.Writer.Header().Set("Access-Control-Expose-Headers", "Link")
	gc.Writer.Header()["Link"] = ss.getLinkHeader()
	gc.Writer.WriteHeader(http.StatusNoContent)
}

func (ss *SignalServer) handleRequest(gc *gin.Context) {
	secret := gc.Param("secret")
	if secret == "" {
		switch gc.Request.Method {
		case http.MethodOptions:
			ss.handleOptions(gc)
		case http.MethodPost:
			ss.handlePost(gc)
		default:
			gc.JSON(http.StatusMethodNotAllowed, gin.H{"error": "method not allowed"})
		}
	} else {
		switch gc.Request.Method {
		case http.MethodPatch:
			ss.handlePatch(gc)
		case http.MethodDelete:
			ss.handleDelete(gc)
		default:
			gc.JSON(http.StatusMethodNotAllowed, gin.H{"error": "method not allowed"})
		}
	}
}

func (ss *SignalServer) handlePost(gc *gin.Context) {
	typ := gc.Param(PathVarType)
	app := gc.Param(PathVarApp)
	stream := gc.Param(PathVarStream)
	if typ == "" || app == "" || stream == "" {
		ss.logger.Errorf("bad request: %s %s %s", typ, app, stream)
		gc.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}

	if typ == "whip" {
		ss.handlePostWhip(gc, app, stream)
	} else {
		ss.handlePostWhep(gc, app, stream)
	}
}

func sessionLocation(publish bool, secret uuid.UUID) string {
	ret := ""
	if publish {
		ret += "whip"
	} else {
		ret += "whep"
	}
	ret += "/" + secret.String()
	return ret
}

func (ss *SignalServer) handlePostWhip(gc *gin.Context, app, stream string) error {

	peerID := guid.S()

	logger := ss.logger.WithFields(logrus.Fields{
		"session": peerID,
		"stream":  stream,
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
		RouterID:   stream,
		Domain:     domain,
		URI:        gc.Request.URL.Path,
		Producer:   true,
	}, logger)

	sdpOffer, err := io.ReadAll(gc.Request.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read sdp offer")
	}

	lsdp, err := s.Publish(2*time.Second, string(sdpOffer))
	if err != nil {
		logger.WithError(err).Error("failed to publish")
		return errors.Wrap(err, "failed to publish")
	}

	logger.WithField("answer", lsdp.SDP).Debug("resp answer")
	gc.Writer.Header().Set("Content-Type", "application/sdp")
	gc.Writer.Header().Set("Access-Control-Expose-Headers", "ETag, ID, Accept-Patch, Link, Location")
	gc.Writer.Header().Set("ETag", "*")
	gc.Writer.Header().Set("ID", peerID)
	gc.Writer.Header().Set("Accept-Patch", "application/trickle-ice-sdpfrag")
	gc.Writer.Header().Set("Location", sessionLocation(true, uuid.New()))
	gc.Writer.Header()["Link"] = ss.getLinkHeader()

	gc.String(http.StatusCreated, lsdp.SDP)

	return nil
}

func (ss *SignalServer) handlePostWhep(gc *gin.Context, app, stream string) error {

	peerID := guid.S()

	logger := ss.logger.WithFields(logrus.Fields{
		"session": peerID,
		"stream":  stream,
	})

	s := rtc.NewServSession(ss.ctx, inter_rtc.StreamFactory(), router.PeerParams{
		RemoteAddr: gc.Request.RemoteAddr,
		LocalAddr:  gc.Request.Host,
		PeerID:     peerID,
		RouterID:   stream,
		Domain:     gc.Request.Host,
		URI:        gc.Request.URL.Path,
		Producer:   true,
	}, logger)

	sdpOffer, err := io.ReadAll(gc.Request.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read sdp offer")
	}

	lsdp, err := s.Subscribe(string(sdpOffer), 4*time.Second)
	if err != nil {
		logger.WithError(err).Error("failed to whep")
		return errors.Wrap(err, "failed to whep")
	}

	logger.WithField("answer", lsdp.SDP).Debug("resp answer")

	gc.String(http.StatusOK, lsdp.SDP)

	return nil
}

func (ss *SignalServer) handlePatch(gc *gin.Context) {
	ss.logger.Infof("handlePatch")
}

func (ss *SignalServer) handleDelete(gc *gin.Context) {
}
