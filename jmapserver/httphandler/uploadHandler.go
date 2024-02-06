package httphandler

import "net/http"

type UploadHandler struct {
}

func NewUploadHandler() UploadHandler {
	return UploadHandler{}
}

func (uh UploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	AddCORSAllowedOriginHeader(w, r)
}

/*
	uploadREStr := strings.ReplaceAll(uploadPath, datatyper.UrlTemplateAccountID, "(\\d+)")
	uploadREStr = escapeCommon(uploadREStr)
	if uploadPathRE, err := regexp.Compile(uploadREStr); err == nil && uploadPathRE.MatchString(p) {
		return handlerTypeUpload
	}
*/
