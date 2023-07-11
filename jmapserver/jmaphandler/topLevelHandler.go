package jmaphandler

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/mjl-/mox/jmapserver/capabilitier"
	"github.com/mjl-/mox/jmapserver/core"
	"github.com/mjl-/mox/jmapserver/datatyper"
	"github.com/mjl-/mox/mlog"
	"github.com/mjl-/mox/store"
)

type JMAPServerHandler struct {
	//Path is the absolute path of the jmap handler as set in http/web.go
	Path string

	//Hostname is the hostname that we are listening on. This is send in the session handler
	Hostname string

	//Port is the port that we are listening on. This is send in the session handler
	Port int

	Capability        []capabilitier.Capabilitier
	OpenEmailAuthFunc OpenEmailAuthFunc
	Logger            *mlog.Log
}

func NewHandler(hostname, path string, port int, openEmailAuthFunc OpenEmailAuthFunc, logger *mlog.Log) JMAPServerHandler {
	return JMAPServerHandler{
		Hostname: hostname,
		Port:     port,
		Path:     path,
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
		OpenEmailAuthFunc: openEmailAuthFunc,
		Logger:            logger,
	}
}

type OpenEmailAuthFunc func(email, password string) (*store.Account, error)

type AuthenticationMiddleware struct {
	OpenEmailAuthFunc OpenEmailAuthFunc
	Logger            *mlog.Log
	contextUserKey    string
}

func NewAuthenticationMiddleware(openEmailAccountFunc OpenEmailAuthFunc, logger *mlog.Log) AuthenticationMiddleware {
	return AuthenticationMiddleware{
		OpenEmailAuthFunc: openEmailAccountFunc,
		Logger:            logger,
		contextUserKey:    defaultContextUserKey,
	}
}

func (authM AuthenticationMiddleware) Authenticate(hf http.Handler) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		//user basic authentication for now
		username, password, ok := r.BasicAuth()
		if !ok {
			//reply with the correct header
			rw.Header().Add("WWW-Authenticate", "Basic realm=\"Authenticate in order to use JMAP\"")
			rw.WriteHeader(http.StatusUnauthorized)
			rw.Write(nil)
			return
		}
		_, err := authM.OpenEmailAuthFunc(username, password)
		if err != nil {
			//there is no details in the spec what needs to send when the authentication fails
			rw.WriteHeader(http.StatusUnauthorized)
			rw.Write([]byte("incorrect/username password"))
			return
		}

		authM.Logger.Debug("login successfull")
		ctx := r.Context()
		ctx = context.WithValue(ctx, authM.contextUserKey, User{
			Username: username,
		})

		hf.ServeHTTP(rw, r.WithContext(ctx))
	}
}

func (jh JMAPServerHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {

	//instantiate subhandlers
	sessionCapabiltyInfo := map[string]interface{}{}

	for _, capability := range jh.Capability {
		jh.Logger.Debug("adding capability", mlog.Field("urn", capability.Urn()))
		sessionCapabiltyInfo[capability.Urn()] = capability.SessionObjectInfo()
	}

	//find out what we were called from because we need this information in the sessionHandler
	baseURL := fmt.Sprintf("https://%s:%d%s", jh.Hostname, jh.Port, jh.Path)

	jh.Logger.Debug("log url", mlog.Field("base-url", baseURL))

	// ../../rfc/8620:679
	sessionPath := jh.Path + "session"
	apiPath := "api"
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
		jh.Logger,
	)

	apiHandler := APIHandler{}
	downloadHandler := DownloadHandler{}
	uploadHandler := UploadHandler{}
	eventSourceHandler := EventSourceHandler{}

	var getRejectUnsupportedMethodsHandler = func(acceptedMethods []string, nextHandler http.Handler) func(resp http.ResponseWriter, req *http.Request) {
		return func(resp http.ResponseWriter, req *http.Request) {
			var methodAccepted bool
			for _, acceptedMethod := range acceptedMethods {
				if req.Method == acceptedMethod {
					methodAccepted = true
					break
				}
			}
			if methodAccepted {
				nextHandler.ServeHTTP(resp, req)
				return
			}
			resp.WriteHeader(http.StatusMethodNotAllowed)
		}
	}

	authMW := NewAuthenticationMiddleware(store.OpenEmailAuth, jh.Logger)

	//create a new mux for routing in this path
	mux := http.NewServeMux()
	mux.HandleFunc(sessionPath, getRejectUnsupportedMethodsHandler([]string{http.MethodGet}, authMW.Authenticate(sessionHandler)))
	jh.Logger.Debug("register path", mlog.Field("sessionPath", sessionPath))
	mux.HandleFunc(apiPath, getRejectUnsupportedMethodsHandler([]string{http.MethodPost}, authMW.Authenticate(apiHandler)))

	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		//do dome pattern matching here
		/*
			go over the remaing candidates and see if we have match
			substitute the templates with a wildcard which matches
		*/

		switch getHandlerForPath(req.URL.Path, downloadPath, uploadPath, eventSourcePath) {
		case handlerTypeDownload:
			getRejectUnsupportedMethodsHandler([]string{http.MethodGet}, authMW.Authenticate(downloadHandler))
			return
		case handlerTypeUpload:
			getRejectUnsupportedMethodsHandler([]string{http.MethodPost}, authMW.Authenticate(uploadHandler))
			return
		case handlerTypeEventSource:
			getRejectUnsupportedMethodsHandler([]string{http.MethodGet}, authMW.Authenticate(eventSourceHandler))
			return
		}

		//if nothing matches, we exit here
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("page not found"))
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
	//FIXME not sure if sending a 404 is fine if the path itself exist but icm parameter values does not make sense. Need to check the spec for that
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
