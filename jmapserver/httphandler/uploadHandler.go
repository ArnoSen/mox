package httphandler

import (
	"net/http"

	"github.com/mjl-/mox/mlog"
)

type UploadHandler struct {
	logger mlog.Log
}

func NewUploadHandler(logger mlog.Log) UploadHandler {
	return UploadHandler{
		logger: logger,
	}
}

func (uh UploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	uh.logger.Debug("upload handler called")
	AddCORSAllowedOriginHeader(w, r)
}

/*
	uploadREStr := strings.ReplaceAll(uploadPath, datatyper.UrlTemplateAccountID, "(\\d+)")
	uploadREStr = escapeCommon(uploadREStr)
	if uploadPathRE, err := regexp.Compile(uploadREStr); err == nil && uploadPathRE.MatchString(p) {
		return handlerTypeUpload
	}
*/
