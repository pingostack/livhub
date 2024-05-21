package httpserv

import (
	"net/http"

	"github.com/sirupsen/logrus"
)

type headerHandler struct {
	http.Handler
	headers map[string]string
	logger  *logrus.Entry
}

func (h *headerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", "Neon")
	for k, v := range h.headers {
		w.Header().Set(k, v)
	}

	h.Handler.ServeHTTP(w, r)
}
