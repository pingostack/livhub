package httpserv

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const (
	defaultMaxAgeSecond = 12 * 60 * 60
)

type HttpParams struct {
	HttpAddr         string            `json:"httpAddr" mapstructure:"httpAddr"`
	HttpsAddr        string            `json:"httpsAddr" mapstructure:"httpsAddr"`
	Cert             string            `json:"cert" mapstructure:"cert"`
	Key              string            `json:"key" mapstructure:"key"`
	AllowOrigin      []string          `json:"allowOrigin" mapstructure:"allowOrigin"`
	AllowMethods     []string          `json:"allowMethods" mapstructure:"allowMethods"`
	AllowHeaders     []string          `json:"AllowHeaders" mapstructure:"AllowHeaders"`
	ExposeHeaders    []string          `json:"exposeHeaders" mapstructure:"exposeHeaders"`
	AllowCredentials bool              `json:"allowCredentials" mapstructure:"allowCredentials"`
	MaxAge           int               `json:"maxAge" mapstructure:"maxAge"`
	Headers          map[string]string `json:"headers" mapstructure:"headers"`
	AllowOriginHook  string            `json:"allowOriginHook" mapstructure:"allowOriginHook"`
}

type SignalServer struct {
	l         *logrus.Entry
	ctx       context.Context
	params    HttpParams
	httpServ  *Server
	httpsServ *Server
}

func NewSignalServer(ctx context.Context, params HttpParams, logger *logrus.Entry) *SignalServer {
	return &SignalServer{
		l:      logger,
		ctx:    ctx,
		params: params,
	}
}

func (ss *SignalServer) validate() error {
	if ss.params.HttpAddr == "" && ss.params.HttpsAddr == "" {
		return errors.New("httpAddr and httpsAddr can't be both empty")
	}

	if ss.params.HttpsAddr != "" && (ss.params.Cert == "" || ss.params.Key == "") {
		return errors.New("cert and key can't be empty when httpsAddr is not empty")
	}

	if len(ss.params.AllowOrigin) == 0 {
		ss.params.AllowOrigin = []string{"*"}
	}

	if len(ss.params.AllowMethods) == 0 {
		ss.params.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"}
	}

	if len(ss.params.AllowHeaders) == 0 {
		ss.params.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "If-Match"}
	}

	if len(ss.params.ExposeHeaders) == 0 {
		ss.params.ExposeHeaders = []string{"Content-Length", "Content-Type"}
	}

	if ss.params.MaxAge == 0 {
		ss.params.MaxAge = defaultMaxAgeSecond
	}

	return nil
}

func (ss *SignalServer) Start(handles ...gin.HandlerFunc) error {
	if err := ss.validate(); err != nil {
		return err
	}

	router := gin.New()
	//router.SetTrustedProxies()
	router.NoRoute(handles...)
	corsConfig := cors.Config{
		AllowOrigins:     ss.params.AllowOrigin,
		AllowMethods:     ss.params.AllowMethods,
		AllowHeaders:     ss.params.AllowHeaders,
		ExposeHeaders:    ss.params.ExposeHeaders,
		AllowCredentials: ss.params.AllowCredentials,
		MaxAge:           time.Duration(ss.params.MaxAge) * time.Second,
	}

	if ss.params.AllowOriginHook != "" {
		corsConfig.AllowOriginFunc = ss.allowOriginHook
	}

	router.Use(cors.New(corsConfig))

	if ss.params.HttpAddr != "" {
		ln, err := net.Listen("tcp", ss.params.HttpAddr)
		if err != nil {
			ss.l.WithError(err).Errorf("http server listen on %s failed", ss.params.HttpAddr)
			panic(err)
		}

		ss.l.Infof("http server listen on %s", ss.params.HttpAddr)

		ss.httpServ = NewServer(ss.ctx,
			router,
			WithListener(ln),
			WithHeaders(ss.params.Headers),
			WithLogger(ss.l))
	}

	if ss.params.HttpsAddr != "" && ss.params.Cert != "" && ss.params.Key != "" {
		ln, err := net.Listen("tcp", ss.params.HttpsAddr)
		if err != nil {
			ss.l.WithError(err).Errorf("https server listen on %s failed", ss.params.HttpsAddr)
			panic(err)
		}

		ss.l.Infof("https server listen on %s", ss.params.HttpsAddr)

		ss.httpsServ = NewServer(ss.ctx,
			router,
			WithListener(ln),
			WithHeaders(ss.params.Headers),
			WithSSL(ss.params.Cert, ss.params.Key),
			WithLogger(ss.l))
	}

	return nil
}

func (ss *SignalServer) Close() error {
	if ss.httpServ != nil {
		ss.httpServ.Close()
	}

	if ss.httpsServ != nil {
		ss.httpsServ.Close()
	}
	return nil
}

func (ss *SignalServer) allowOriginHook(origin string) bool {
	return true
}
