package httpserv

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/pingostack/neon/pkg/logger"
)

type loggerWriter struct {
	w      http.ResponseWriter
	status int
	buf    bytes.Buffer
}

func (w *loggerWriter) Header() http.Header {
	return w.w.Header()
}

func (w *loggerWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	w.buf.Write(b)
	return w.w.Write(b)
}

func (w *loggerWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.w.WriteHeader(statusCode)
}

func (w *loggerWriter) dump() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "HTTP/1.1 %d %s\n", w.status, http.StatusText(w.status))
	w.Header().Write(&buf)
	buf.Write([]byte("\n"))
	if w.buf.Len() > 0 {
		fmt.Fprintf(&buf, "(body of %d bytes)\n", w.buf.Len())
	}

	return buf.String()
}

type loggerHandler struct {
	logger logger.Logger
	http.Handler
}

func (h *loggerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.logger.Debugf("[conn %v] %s %s", r.RemoteAddr, r.Method, r.URL)

	bytes, _ := httputil.DumpRequest(r, true)
	h.logger.Debugf("[conn %v] request: %s", r.RemoteAddr, string(bytes))

	lw := &loggerWriter{
		w: w,
	}

	h.Handler.ServeHTTP(lw, r)

	h.logger.Debugf("[conn %v] response: %s", r.RemoteAddr, lw.dump())
}
