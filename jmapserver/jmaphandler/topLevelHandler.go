package jmaphandler

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/mjl-/mox/jmapserver/capabilitier"
	"github.com/mjl-/mox/jmapserver/core"
	"github.com/mjl-/mox/jmapserver/datatyper"
)

type JMAPServerHandler struct {
	//Path is the absolute path of the jmap handler as set in http/web.go
	Path       string
	Capability []capabilitier.Capabilitier
}

func NewHandler(path string) JMAPServerHandler {
	return JMAPServerHandler{
		Path: path,
		Capability: []capabilitier.Capabilitier{
			core.NewCore(core.CoreCapabilitySettings{
				// ../../rfc/8620:517
				//use the minimum recommneded values for now. Maybe move some to settings later on
				MaxSizeUpload:         50000000,
				MaxConcurrentUpload:   4,
				MaxSizeRequest:        10000000,
				MaxConcurrentRequests: 4,
				MaxCallsInRequest:     16,
				MaxObjectsInGet:       500,
				MaxObjectsInSet:       500,
				CollationAlgorithms: []string{
					// ../../rfc/4790:1127
					//not sure yet how this works out later on but let's put in some basic value
					"i;ascii-casemap",
				},
			}),
		},
	}
}

func authenticationMiddleware(hf http.Handler) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		//user basic authentication for now

		hf.ServeHTTP(rw, r)
	}
}

func (jh JMAPServerHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {

	//instantiate subhandlers
	var sessionCapabiltyInfo map[string]interface{}

	for _, capability := range jh.Capability {
		sessionCapabiltyInfo[capability.Urn()] = capability.SessionObjectInfo()
	}

	//find out what we were called from because we need this information in the sessionHandler
	baseURL := fmt.Sprintf("%s%s:%s%s", r.URL.Scheme, r.URL.Host, r.URL.Port(), jh.Path)

	// ../../rfc/8620:679
	sessionPath := baseURL + "session"
	apiPath := baseURL + "api"
	downloadPath := fmt.Sprintf("download/%s/%s/%s/%s", datatyper.UrlTemplateAccountID, datatyper.UrlTemplateBlodId, datatyper.UrlTemplateType, datatyper.UrlTemplateName)
	uploadPath := fmt.Sprintf("upload/%s", datatyper.UrlTemplateAccountID)
	eventSourcePath := fmt.Sprintf("eventsource/?types=%s&closeafter=%s&ping=%s", datatyper.UrlTemplateTypes, datatyper.UrlTemplateClosedAfter, datatyper.UrlTemplatePing)

	sessionHandler := NewSessionHandler(
		nil, //FIXME need a valid object here
		sessionCapabiltyInfo,
		baseURL+apiPath,
		baseURL+downloadPath,
		baseURL+uploadPath,
		baseURL+eventSourcePath,
	)

	apiHandler := APIHandler{}
	downloadHandler := DownloadHandler{}
	uploadHandler := UploadHandler{}
	eventSourceHandler := EventSourceHandler{}

	var getRejectUnsupportedMethodsHandler = func(acceptedMethods []string, nextHandler http.Handler) func(resp http.ResponseWriter, req *http.Request) {
		return func(resp http.ResponseWriter, req *http.Request) {
			for _, acceptedMethod := range acceptedMethods {
				if req.Method == acceptedMethod {
					nextHandler.ServeHTTP(resp, req)
				}
			}
			resp.WriteHeader(http.StatusMethodNotAllowed)
		}
	}

	//create a new mux for routing in this path
	mux := http.NewServeMux()
	mux.HandleFunc(sessionPath, getRejectUnsupportedMethodsHandler([]string{http.MethodGet}, authenticationMiddleware(sessionHandler)))
	mux.HandleFunc(apiPath, getRejectUnsupportedMethodsHandler([]string{http.MethodPost}, authenticationMiddleware(apiHandler)))

	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		//do dome pattern matching here
		/*
			go over the remaing candidates and see if we have match
			substitute the templates with a wildcard which matches
		*/

		switch getHandlerForPath(req.URL.Path, downloadPath, uploadPath, eventSourcePath) {
		case handlerTypeDownload:
			getRejectUnsupportedMethodsHandler([]string{http.MethodGet}, authenticationMiddleware(downloadHandler))
			return
		case handlerTypeUpload:
			getRejectUnsupportedMethodsHandler([]string{http.MethodPost}, authenticationMiddleware(uploadHandler))
			return
		case handlerTypeEventSource:
			getRejectUnsupportedMethodsHandler([]string{http.MethodGet}, authenticationMiddleware(eventSourceHandler))
			return
		}

		//if nothing matches, we exit here
		w.WriteHeader(http.StatusNotFound)
		return
	})

	mux.ServeHTTP(rw, r)
}

type handlerType int

const (
	handlerTypeUndefined handlerType = iota
	handlerTypeDownload
	handlerTypeUpload
	handlerTypeEventSource
)

func getHandlerForPath(p, downloadPath, uploadPath, eventSourcePath string) handlerType {
	//FIXME not sure if sending a 404 is fine if the path itself exist but icm paramter values does not make sense. Need to check the spec for that
	//FIXME mabye this should be split up in 3 different fns but on the other side only a routing decision needs to be taken

	var escapeCommon = func(s string) string {
		result := strings.ReplaceAll(s, "?", "\\?")
		return result
	}

	//replace the placeholders with a none empty wildcard
	downloadPathREStr := strings.ReplaceAll(downloadPath, datatyper.UrlTemplateAccountID, "(\\d+)")
	downloadPathREStr = strings.ReplaceAll(downloadPathREStr, datatyper.UrlTemplateBlodId, "(\\d+)")
	downloadPathREStr = strings.ReplaceAll(downloadPathREStr, datatyper.UrlTemplateName, "(\\S+)")
	downloadPathREStr = strings.ReplaceAll(downloadPathREStr, datatyper.UrlTemplateType, "(\\S+)")
	downloadPathREStr = escapeCommon(downloadPathREStr)
	if downloadPathRE, err := regexp.Compile(downloadPathREStr); err == nil && downloadPathRE.MatchString(p) {
		return handlerTypeDownload
	}

	uploadREStr := strings.ReplaceAll(uploadPath, datatyper.UrlTemplateAccountID, "(\\d+)")
	uploadREStr = escapeCommon(uploadREStr)
	if uploadPathRE, err := regexp.Compile(uploadREStr); err == nil && uploadPathRE.MatchString(p) {
		return handlerTypeUpload
	}

	eventSourceREStr := strings.ReplaceAll(eventSourcePath, datatyper.UrlTemplateTypes, "(\\S+)")
	eventSourceREStr = strings.ReplaceAll(eventSourceREStr, datatyper.UrlTemplateClosedAfter, "(\\d+)")
	eventSourceREStr = strings.ReplaceAll(eventSourceREStr, datatyper.UrlTemplatePing, "(\\d+)")
	eventSourceREStr = escapeCommon(eventSourceREStr)

	if eventSourcePathRE, err := regexp.Compile(eventSourceREStr); err == nil && eventSourcePathRE.MatchString(p) {
		return handlerTypeEventSource
	}

	return handlerTypeUndefined
}
