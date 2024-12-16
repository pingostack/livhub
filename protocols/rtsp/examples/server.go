package main

import (
	"fmt"
	"os"
	"time"

	"github.com/pingostack/livhub/protocols/rtsp"
	"github.com/sirupsen/logrus"
)

type TestServer struct {
	logger *logrus.Logger
}
type TestSession struct {
	*rtsp.ServSession
	logger *logrus.Entry
}

func (s *TestSession) Logger() rtsp.Logger {
	return s.logger
}

func (ts *TestServer) OnConnect(ss rtsp.IServSession) {
	fmt.Println("connect")
}

func (ts *TestServer) OnDisconnect(ss rtsp.IServSession) {
	fmt.Println("disconnect")
}

func (ts *TestServer) OnShutdown(s *rtsp.Server) {
	fmt.Println("server shutdown")
}

func (ts *TestServer) OnDescribe(serv *rtsp.Serv) error {
	fmt.Println("describe")
	return nil
}

func (ts *TestServer) OnAnnounce(serv *rtsp.Serv) error {
	fmt.Println("announce")

	return nil
}

func (ts *TestServer) OnPause(serv *rtsp.Serv) error {
	fmt.Println("pause")
	return nil
}

func (ts *TestServer) OnResume(serv *rtsp.Serv) error {
	fmt.Println("resume")
	return nil
}

func (ts *TestServer) OnStream(serv *rtsp.Serv) error {
	fmt.Println("stream")
	return nil
}

func (ts *TestServer) NewOrGet() rtsp.IServSession {
	return &TestSession{
		ServSession: rtsp.NewServSession(ts),
		logger:      ts.logger.WithFields(logrus.Fields{"session": "test"}),
	}
}

func main() {
	log := logrus.New()
	ts := &TestServer{
		logger: log,
	}
	log.SetLevel(logrus.DebugLevel)
	log.SetOutput(os.Stdout)
	s, err := rtsp.NewServer(ts, ts, ":8554", rtsp.Options{
		ReusePort:        true,
		ReuseAddr:        true,
		TCPKeepAlive:     10 * time.Second,
		TCPNoDelay:       true,
		LockOSThread:     true,
		SocketRecvBuffer: 1024 * 1024,
		SocketSendBuffer: 1024 * 1024,
		Logger:           log,
		Multicore:        true,
		NumEventLoop:     0,
	})
	if err != nil {
		panic(err)
	}
	s.Run()
}
