package buffer

import (
	"io"
	"sync/atomic"

	"github.com/let-light/neon/pkg/utils"
)

type RTCPReader struct {
	ssrc     uint32
	closed   utils.AtomicBool
	onPacket atomic.Value //func([]byte)
	onClose  func()
}

func NewRTCPReader(ssrc uint32) *RTCPReader {
	return &RTCPReader{ssrc: ssrc}
}

func (r *RTCPReader) Write(p []byte) (n int, err error) {
	if r.closed.Get() {
		err = io.EOF
		return
	}
	if f, ok := r.onPacket.Load().(func([]byte)); ok {
		f(p)
	}
	return
}

func (r *RTCPReader) OnClose(fn func()) {
	r.onClose = fn
}

func (r *RTCPReader) Close() error {
	r.closed.Set(true)
	r.onClose()
	return nil
}

func (r *RTCPReader) OnPacket(f func([]byte)) {
	r.onPacket.Store(f)
}

func (r *RTCPReader) Read(_ []byte) (n int, err error) { return }
