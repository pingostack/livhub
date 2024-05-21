package rtsp

import (
	"context"
	"fmt"

	"github.com/panjf2000/gnet"
)

type servConn struct {
	*Serv
	c gnet.Conn
}

type Server struct {
	gnet.EventServer
	eventListener IServerEventListener
	provider      ISessionProvider
	opt           Options
	addr          string
}

func NewServer(eventListener IServerEventListener, provider ISessionProvider, addr string, opt Options) (*Server, error) {
	s := &Server{
		eventListener: eventListener,
		provider:      provider,
		opt:           opt,
		addr:          addr,
	}

	if opt.Logger == nil {
		return nil, fmt.Errorf("logger is nil")
	}

	return s, nil
}

func (s *Server) Run() error {
	opt := gnet.Options{
		ReusePort:        s.opt.ReusePort,
		ReuseAddr:        s.opt.ReuseAddr,
		TCPKeepAlive:     s.opt.TCPKeepAlive,
		SocketRecvBuffer: s.opt.SocketRecvBuffer,
		SocketSendBuffer: s.opt.SocketSendBuffer,
		Logger:           s.opt.Logger,
		Multicore:        s.opt.Multicore,
		NumEventLoop:     s.opt.NumEventLoop,
		Codec:            s,
	}

	s.opt.Logger.Infof("server is running on %s", s.addr)
	err := gnet.Serve(s, s.addr, gnet.WithOptions(opt))
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return gnet.Stop(ctx, s.addr)
}

func (s *Server) OnInitComplete(gs gnet.Server) (action gnet.Action) {
	return
}

func (s *Server) OnShutdown(gs gnet.Server) {
	if s.eventListener != nil {
		s.eventListener.OnShutdown(s)
	}
}

func (s *Server) OnOpened(c gnet.Conn) (out []byte, action gnet.Action) {
	session := s.provider.NewOrGet()
	sc := &servConn{
		Serv: NewServ(session, ServOptions{
			Logger:      session.Logger(),
			IdleTimeout: s.opt.IdleTimeout,
			Write: func(data []byte) error {
				return c.AsyncWrite(data)
			},
		}),
		c: c,
	}

	session.AddParams(s, sc)
	c.SetContext(session)

	return
}

func (s *Server) OnClosed(c gnet.Conn, err error) (action gnet.Action) {
	if s.eventListener != nil {
		ss, err := s.getServSession(c)
		if err == nil {
			s.eventListener.OnDisconnect(ss)
		}
	}

	return
}

func (s *Server) Encode(c gnet.Conn, buf []byte) ([]byte, error) {
	return buf, nil
}

func (s *Server) Decode(c gnet.Conn) ([]byte, error) {
	sc, err := s.getServConn(c)
	if err != nil {
		s.opt.Logger.Errorf("getServSession error: %v", err)
		return nil, err
	}

	offset, err := sc.Serv.Feed(c.Read())
	if offset > 0 && offset < c.BufferLength() {
		c.ShiftN(offset)
	} else if offset >= c.BufferLength() {
		c.ResetBuffer()
	}

	if err != nil {
		s.opt.Logger.Errorf("serv feed error: %v", err)
		return nil, err
	}

	return nil, nil
}

func (s *Server) getServConn(c gnet.Conn) (*servConn, error) {
	session, err := s.getServSession(c)
	if err != nil {
		s.opt.Logger.Errorf("getServSession error: %v", err)
		return nil, err
	}

	conn, found := session.GetParams(s)
	if !found {
		s.opt.Logger.Errorf("connection is not found")
		return nil, fmt.Errorf("connection is not found")
	}

	return conn.(*servConn), nil
}

func (s *Server) getServSession(c gnet.Conn) (IServSession, error) {
	cctx := c.Context()
	if cctx == nil {
		s.opt.Logger.Errorf("connection context is nil")
		return nil, fmt.Errorf("connection context is nil")
	}

	session := cctx.(IServSession)
	if session == nil {
		s.opt.Logger.Errorf("session is nil")
		return nil, fmt.Errorf("session is nil")
	}

	return session, nil
}
