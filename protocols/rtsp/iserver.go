package rtsp

import (
	"time"
)

type IServerEventListener interface {
	OnShutdown(s *Server)
	OnConnect(ss IServSession)
	OnDisconnect(ss IServSession)
}

type ISessionProvider interface {
	NewOrGet() IServSession
}

type Options struct {

	// ReuseAddr indicates whether to set up the SO_REUSEADDR socket option.
	ReuseAddr bool

	// ReusePort indicates whether to set up the SO_REUSEPORT socket option.
	ReusePort bool

	// TCPKeepAlive sets up a duration for (SO_KEEPALIVE) socket option.
	TCPKeepAlive time.Duration

	// TCPNoDelay enables/disables the TCP_NODELAY socket option.
	TCPNoDelay bool

	// LockOSThread enables/disables the runtime.LockOSThread() call.
	LockOSThread bool

	// SocketRecvBuffer sets the maximum socket receive buffer in bytes.
	SocketRecvBuffer int

	// SocketSendBuffer sets the maximum socket send buffer in bytes.
	SocketSendBuffer int

	// Logger is the logger for the server.
	Logger Logger

	// NumEventLoop is the number of event loops.
	NumEventLoop int

	// Multicore enables/disables the multi-core execution.
	Multicore bool

	// IdleTimeout is the maximum duration for the connection to be idle.
	IdleTimeout time.Duration
}
