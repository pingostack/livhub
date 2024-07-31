package whip

import (
	"context"
	"fmt"
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

const (
	PathVarType   = "type"
	PathVarApp    = "app"
	PathVarStream = "stream"
	PathVarSecret = "secret"
)

var (
	// path for post and options
	PathPost = fmt.Sprintf("/:%s(%s|%s)/:%s/:%s", PathVarType, "whip", "whep", PathVarApp, PathVarStream)
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

	ss.ss.DefaultRouter().Any(PathPost, ss.handleRequest)
	ss.ss.DefaultRouter().Any("/:type(whip|whep)/:app/:stream/:secret", ss.handleRequest)

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

func (ss *SignalServer) handleOptions(gc *gin.Context) {
	gc.Writer.Header().Set("Access-Control-Allow-Methods", "OPTIONS, GET, POST, PATCH, DELETE")
	gc.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, If-Match")
	gc.Writer.Header().Set("Access-Control-Expose-Headers", "Link")
	gc.Writer.Header()["Link"] = []string{"<https://www.w3.org/TR/webrtc/>; rel=\"help\"", "<https://www.w3.org/TR/webrtc/>; rel=\"help\""}
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

}

func (ss *SignalServer) handlePatch(gc *gin.Context) {
}

func (ss *SignalServer) handleDelete(gc *gin.Context) {
}
