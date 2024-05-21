//go:build !windows
// +build !windows

package config

import (
	"net"
	"syscall"

	"github.com/pingostack/neon/pkg/logger"
)

func init() {
	val, err := getUDPReadBuffer()
	if err == nil {
		if val < minUDPBufferSize {
			logger.Warn("UDP receive buffer is too small for a production set-up", nil,
				"current", val,
				"suggested", minUDPBufferSize)
		} else {
			logger.Debug("UDP receive buffer size", "current", val)
		}
	}
}

func getUDPReadBuffer() (int, error) {
	conn, err := net.ListenUDP("udp4", nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetReadBuffer(defaultUDPBufferSize)
	fd, err := conn.File()
	if err != nil {
		return 0, nil
	}
	defer func() { _ = fd.Close() }()

	return syscall.GetsockoptInt(int(fd.Fd()), syscall.SOL_SOCKET, syscall.SO_RCVBUF)
}
