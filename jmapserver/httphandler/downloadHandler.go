package httphandler

import "net/http"

type DownloadHandler struct {
}

func (dh DownloadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	AddCORSAllowedOriginHeader(w, r)
}
