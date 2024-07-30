package whip

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-contrib/cors"
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
	// 配置 CORS 中间件
	corsConfig := cors.Config{
		AllowOrigins:     []string{"*"}, // 允许所有来源
		AllowMethods:     []string{"OPTIONS", "GET", "POST", "PATCH", "DELETE"},
		AllowHeaders:     []string{"Content-Type", "Authorization", "If-Match"},
		ExposeHeaders:    []string{"Link"},
		AllowCredentials: true,
	}

	ss.ss.DefaultRouter().Use(cors.New(corsConfig))

	ss.ss.DefaultRouter().Use(func(gc *gin.Context) {
		gc.Writer.Header().Set("Access-Control-Allow-Origin", strings.Join(ss.httpParams.AllowOrigin, " ,"))
		gc.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	})

	ss.ss.DefaultRouter().POST("/whip/:app/:stream/", ss.handlePostWhip)
	ss.ss.DefaultRouter().OPTIONS("/whip/:app/:stream/:method", ss.handleOptions)
	ss.ss.DefaultRouter().PATCH("/whip/:app/:stream/:method/:secret", ss.handlePatchWhip)
	ss.ss.DefaultRouter().DELETE("/whip/:app/:stream/:method/:secret", ss.handleDeleteWhip)

	ss.ss.DefaultRouter().POST("/whep/:app/:stream/", ss.handlePostWhep)
	ss.ss.DefaultRouter().OPTIONS("/whep/:app/:stream/:method", ss.handleOptions)
	ss.ss.DefaultRouter().PATCH("/whep/:app/:stream/:method/:secret", ss.handlePatchWhep)
	ss.ss.DefaultRouter().DELETE("/whep/:app/:stream/:method/:secret", ss.handleDeleteWhep)

	ss.ss.DefaultRouter().OPTIONS("", func(ctx *gin.Context) {
		ctx.Writer.Header().Set("Access-Control-Allow-Methods", "OPTIONS, GET, POST, PATCH, DELETE")
		ctx.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, If-Match")
		ctx.Writer.WriteHeader(http.StatusNoContent)
	})
	return ss.ss.Start()
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

func (ss *SignalServer) handlePatchWhep(gc *gin.Context) {
}

func (ss *SignalServer) handleDeleteWhep(gc *gin.Context) {
}

func (ss *SignalServer) handlePatchWhip(gc *gin.Context) {
}

func (ss *SignalServer) handleDeleteWhip(gc *gin.Context) {
}
