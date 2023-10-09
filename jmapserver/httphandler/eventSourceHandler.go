package httphandler

import "net/http"

type EventSourceHandler struct {
}

func (eh EventSourceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	AddCORSAllowedOriginHeader(w, r)
}
