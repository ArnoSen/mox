package httphandler

import "net/http"

type UploadHandler struct {
}

func (uh UploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	addCORSAllowedOriginHeader(w, r)
}
