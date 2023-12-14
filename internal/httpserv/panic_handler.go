package httpserv

import (
	"fmt"
	"net/http"
	"runtime"

	"github.com/sirupsen/logrus"
)

type panicHandler struct {
	http.Handler
	logger *logrus.Entry
}

func (p *panicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			buf := make([]byte, 1<<16)
			n := runtime.Stack(buf, false)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("panic: %v\n%s", err, buf[:n])))
			p.logger.Errorf("Error during: %v\n%s", err, buf[:n])
		}
	}()

	p.Handler.ServeHTTP(w, r)
}
