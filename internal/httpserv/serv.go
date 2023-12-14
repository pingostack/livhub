package httpserv

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"

	"github.com/pingostack/neon/pkg/logger"
	"github.com/sirupsen/logrus"
)

type Server struct {
	ln        net.Listener
	serv      *http.Server
	tlsConfig *tls.Config
	logger    logger.Logger
	ctx       context.Context
	headers   map[string]string
}

type ServerOption func(*Server)

func WithListener(ln net.Listener) ServerOption {
	return func(s *Server) {
		s.ln = ln
	}
}

func WithSSL(cert, key string) ServerOption {
	return func(s *Server) {
		tlsConfig := &tls.Config{}
		cert, err := tls.LoadX509KeyPair(cert, key)
		if err != nil {
			panic(err)
		}

		tlsConfig.Certificates = []tls.Certificate{cert}

		s.tlsConfig = tlsConfig
	}
}

func WithLogger(logger *logrus.Entry) ServerOption {
	return func(s *Server) {
		s.logger = logger.WithField("internal", "httpserv")
	}
}

func WithHeaders(headers map[string]string) ServerOption {
	return func(s *Server) {
		s.headers = headers
	}
}

func NewServer(ctx context.Context, handler http.Handler, opts ...ServerOption) *Server {
	s := &Server{
		ctx: ctx,
	}

	for _, opt := range opts {
		opt(s)
	}

	h := handler
	h = &loggerHandler{
		logger:  logrus.WithField("handler", "logger"),
		Handler: h,
	}

	h = &headerHandler{
		Handler: h,
		headers: s.headers,
		logger:  logrus.WithField("handler", "header"),
	}

	h = &panicHandler{
		Handler: h,
		logger:  logrus.WithField("handler", "panic"),
	}

	s.serv = &http.Server{
		Handler:   h,
		TLSConfig: s.tlsConfig,
	}

	if s.tlsConfig != nil {
		go s.serv.ServeTLS(s.ln, "", "")
	} else {
		go s.serv.Serve(s.ln)
	}

	return s
}

func (s *Server) Close() {
	if s.serv != nil {
		s.serv.Shutdown(context.Background())
	}

	s.serv.Close()
}
