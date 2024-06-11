package whip

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/let-light/gomodule"
	feature_rtc "github.com/pingostack/neon/features/rtc"
	"github.com/pingostack/neon/internal/httpserv"
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

func (ss *SignalServer) Start() error {
	ss.ss.DefaultRouter().Use(func(gc *gin.Context) {
		gc.Writer.Header().Set("Access-Control-Allow-Origin", strings.Join(ss.httpParams.AllowOrigin, " ,"))
		gc.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	})

	ss.ss.DefaultRouter().POST("/:app/:stream/whip", ss.handlePostWhip)
	ss.ss.DefaultRouter().OPTIONS("/:app/:stream/whip", ss.handleOptions)
	ss.ss.DefaultRouter().POST("/:app/:stream/whep", ss.handlePostWhep)
	ss.ss.DefaultRouter().OPTIONS("/:app/:stream/whep", ss.handleOptions)

	return ss.ss.Start(ss.handleRequest)
}

func (ss *SignalServer) Close() error {
	return ss.ss.Close()
}

func (ss *SignalServer) handleOptions(gc *gin.Context) {
	gc.Writer.Header().Set("Access-Control-Allow-Methods", "OPTIONS, GET, POST, PATCH, DELETE")
	gc.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, If-Match")
	gc.Writer.Header().Set("Access-Control-Expose-Headers", "Link")
	gc.Writer.Header().Set("Link", "")
	gc.Writer.WriteHeader(http.StatusNoContent)
}

func (ss *SignalServer) handlePostWhip(gc *gin.Context) {

}

func (ss *SignalServer) handlePostWhep(gc *gin.Context) {

}

func (ss *SignalServer) handleRequest(gc *gin.Context) {
	if gc.Request.Method == http.MethodOptions &&
		gc.Request.Header.Get("Access-Control-Request-Method") != "" {
		gc.Writer.Header().Set("Access-Control-Allow-Methods", strings.Join(ss.ss.AllowMethods(), ", "))
		gc.Writer.Header().Set("Access-Control-Allow-Headers", strings.Join(ss.ss.AllowHeaders(), ", "))
		gc.Writer.WriteHeader(http.StatusNoContent)
		return
	}
}
