package httphandler

import "net/http"

type DownloadHandler struct {
	pathFormat string
}

func NewDownloadHandler(pathFormat string) *DownloadHandler {
	return &DownloadHandler{
		pathFormat: pathFormat,
	}
}

func (dh DownloadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	AddCORSAllowedOriginHeader(w, r)
}

/*
	var escapeCommon = func(s string) string {
		result := strings.ReplaceAll(s, "?", "\\?")
		return result
	}

	//replace the placeholders with a none empty wildcard
	downloadPathREStr := strings.ReplaceAll(downloadPath, datatyper.UrlTemplateAccountID, "(\\d+)")
	downloadPathREStr = strings.ReplaceAll(downloadPathREStr, datatyper.UrlTemplateBlodId, "(\\S+)")
	downloadPathREStr = strings.ReplaceAll(downloadPathREStr, datatyper.UrlTemplateName, "(\\S+)")
	downloadPathREStr = strings.ReplaceAll(downloadPathREStr, datatyper.UrlTemplateType, "(\\S+)")
	downloadPathREStr = escapeCommon(downloadPathREStr)
	if downloadPathRE, err := regexp.Compile(downloadPathREStr); err == nil && downloadPathRE.MatchString(p) {
		return handlerTypeDownload
	}
*/
