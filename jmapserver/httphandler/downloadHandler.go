package httphandler

import (
	"fmt"
	"net/http"
	"strings"

	"log/slog"

	"github.com/mjl-/mox/jmapserver/jaccount"
	"github.com/mjl-/mox/jmapserver/user"
	"github.com/mjl-/mox/mlog"
)

func NewInvalidDownloadURL(format string) JSONProblem {
	return JSONProblem{
		Title:   "invalid download url",
		Details: "format is %s ",
	}

}

var BlobNotFound = JSONProblem{
	Title: "blob not found",
}

type DownloadHandler struct {
	pathFormat     string
	contextUserKey string
	logger         mlog.Log
	AccountOpener  AccountOpener
}

func NewDownloadHandler(ao AccountOpener, contextUserKey, pathFormat string, logger mlog.Log) *DownloadHandler {
	return &DownloadHandler{
		contextUserKey: contextUserKey,
		pathFormat:     pathFormat,
		logger:         logger,
		AccountOpener:  ao,
	}
}

func (dh DownloadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	dh.logger.Debug("download handler called")
	AddCORSAllowedOriginHeader(w, r)

	//we are authenticated but we need some checks
	contentType := r.URL.Query().Get("type")
	if contentType == "" {
		dh.logger.Error("content type empty")
		sendUserErr(w, TypeCannotBeEmpty)
		return
	}

	//FIXME this is hard coded now and if the format changes this should change as well
	dh.logger.Logger.Debug("url", slog.String("path", r.URL.Path))
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) != 6 {
		//NB: the first element is an empty string because path starts with a '/'
		dh.logger.Error("path must be 5 elements")
		sendUserErr(w, NewInvalidDownloadURL(dh.pathFormat))
		return
	}

	if pathParts[3] != "000" {
		sendUserErr(w, UnknownAccount)
		return
	}

	blobID := pathParts[4]
	name := pathParts[5]

	dh.logger.Debug("parsing download url", slog.Any("blodId", blobID), slog.Any("name", name), slog.Any("type", contentType))

	//pass in the jaccount
	userIface := r.Context().Value(dh.contextUserKey)
	if userIface == nil {
		dh.logger.Error("no user found in context")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	userObj, ok := userIface.(user.User)
	if !ok {
		dh.logger.Error("user is not of type user.User", slog.Any("unexpectedtype", fmt.Sprintf("%T", userIface)))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	mAccount, err := dh.AccountOpener(dh.logger, userObj.Name)
	if err != nil {
		dh.logger.Error("error opening account", slog.Any("err", err.Error()), slog.Any("accountname", userObj.Email))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	found, bytes, err := jaccount.NewJAccount(mAccount, dh.logger).DownloadBlob(r.Context(), blobID, name, contentType)
	if err != nil {
		dh.logger.Error("error opening account", slog.Any("err", err.Error()), slog.Any("accountname", userObj.Email))
		w.WriteHeader(http.StatusInternalServerError)

	}
	if !found {
		dh.logger.Info("blob not found", slog.Any("blodId", blobID))
		sendUserErr(w, BlobNotFound)
		return
	}

	w.Header().Set(HeaderContentType, contentType)
	//FIXME uncomment below line when this part of the code is stable
	//w.Header().Set("Cache-Control", "private, immutable, max-age=31536000")
	if name != "null" {
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s`, name))
	}
	//FIXME need to make this streaming to prevent large memory allocations

	dh.logger.Info("writing bytes", slog.Any("size", len(bytes)))
	w.Write(bytes)

	dh.logger.Debug("download response", slog.Any("bytes", string(bytes)))
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
