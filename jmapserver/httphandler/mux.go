package httphandler

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/mjl-/mox/jmapserver/capabilitier"
	"github.com/mjl-/mox/jmapserver/core"
	"github.com/mjl-/mox/jmapserver/datatyper"
	"github.com/mjl-/mox/jmapserver/mailcapability"
	"github.com/mjl-/mox/jmapserver/user"
	"github.com/mjl-/mox/mlog"
	"github.com/mjl-/mox/store"
	"golang.org/x/exp/slog"
)

const (
	corsAllowOriginCtxKey = "Access-Control-Allow-Origin"
	corsAllowOriginHeader = "Access-Control-Allow-Origin"
	defaultContextUserKey = "_user"
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
	AccountOpener     AccountOpener

	//CORSAllowFrom defines the hosts that can access JMAP resources from a browser
	CORSAllowFrom []string
	Logger        mlog.Log

	contextUserKey string

	sessionCapabilityInfo map[string]interface{}
}

func NewHandler(hostname, path string, port int, openEmailAuthFunc OpenEmailAuthFunc, accountOpener AccountOpener, corsAllowFrom []string, logger mlog.Log) JMAPServerHandler {

	result := JMAPServerHandler{
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
			mailcapability.NewMailCapability(mailcapability.NewDefaultMailCapabilitySettings(), defaultContextUserKey),
		},
		CORSAllowFrom:         corsAllowFrom,
		OpenEmailAuthFunc:     openEmailAuthFunc,
		AccountOpener:         accountOpener,
		Logger:                logger,
		contextUserKey:        defaultContextUserKey,
		sessionCapabilityInfo: make(map[string]interface{}),
	}

	for _, capability := range result.Capability {
		result.sessionCapabilityInfo[capability.Urn()] = capability.SessionObjectInfo()
	}

	return result
}

type OpenEmailAuthFunc func(log mlog.Log, email, password string) (*store.Account, error)

type AuthenticationMiddleware struct {
	OpenEmailAuthFunc OpenEmailAuthFunc
	Logger            mlog.Log
	contextUserKey    string
}

func NewAuthenticationMiddleware(openEmailAccountFunc OpenEmailAuthFunc, logger mlog.Log, contextUserKey string) AuthenticationMiddleware {
	return AuthenticationMiddleware{
		OpenEmailAuthFunc: openEmailAccountFunc,
		Logger:            logger,
		contextUserKey:    contextUserKey,
	}
}

func (authM AuthenticationMiddleware) Authenticate(hf http.Handler) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {

		//user basic authentication for now
		email, password, ok := r.BasicAuth()
		if !ok {
			//reply with the correct header
			rw.Header().Add("WWW-Authenticate", "Basic realm=\"Authenticate in order to use JMAP\"")
			rw.WriteHeader(http.StatusUnauthorized)
			rw.Write(nil)
			return
		}

		//remove the auth header so it does not end up in the logs when we dump requests in debug
		r.Header.Del("Authorization")

		account, err := authM.OpenEmailAuthFunc(authM.Logger, email, password)
		if err != nil {
			//there is no details in the spec what needs to send when the authentication fails
			authM.Logger.Debug(fmt.Sprintf("authentication err %s", err))
			rw.WriteHeader(http.StatusUnauthorized)
			rw.Write([]byte("incorrect/username password"))
			return
		}

		authM.Logger.Debug("login successfull")
		ctx := r.Context()
		ctx = context.WithValue(ctx, authM.contextUserKey, user.User{
			Email: email,
			Name:  account.Name,
		})

		hf.ServeHTTP(rw, r.WithContext(ctx))
	}
}

type CORSMiddleware struct {
	AllowFrom      []string
	HeadersAllowed []string
}

func NewCORSMiddleware(allowFrom, headersAllowed []string) CORSMiddleware {
	return CORSMiddleware{
		AllowFrom:      allowFrom,
		HeadersAllowed: headersAllowed,
	}
}

func (cm CORSMiddleware) HandleMethodOptions(h http.HandlerFunc) http.HandlerFunc {
	//https://fetch.spec.whatwg.org/
	return func(rw http.ResponseWriter, r *http.Request) {

		var finalAllowFrom string
		for i, allowFrom := range cm.AllowFrom {
			if i == 0 {
				finalAllowFrom = allowFrom
			}

			//when there are multiple allows, then we should reply with the origins host
			if allowFrom == r.Host {
				finalAllowFrom = r.Host
			}
		}

		if r.Method == http.MethodOptions {
			if finalAllowFrom != "" {
				rw.Header().Set("Access-Control-Allow-Origin", finalAllowFrom)
				rw.Header().Set("Access-Control-Allow-Headers", strings.Join(cm.HeadersAllowed, ","))
			}
			rw.Write(nil)
			return
		}

		ctx := r.Context()
		if finalAllowFrom != "" {
			//save the cors allow origin host in ctx because we need it later
			ctx = context.WithValue(ctx, corsAllowOriginCtxKey, finalAllowFrom)
		}

		h.ServeHTTP(rw, r.WithContext(ctx))
	}
}

func (jh JMAPServerHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {

	//find out what we were called from because we need this information in the sessionHandler
	baseURL := fmt.Sprintf("https://%s:%d", jh.Hostname, jh.Port)

	jh.Logger.Debug("log url", slog.Any("base-url", baseURL))

	// ../../rfc/8620:679
	sessionPath := jh.Path + "session"
	apiPath := jh.Path + "api"
	downloadPath := fmt.Sprintf("%sdownload/%s/%s/%s/%s", jh.Path, datatyper.UrlTemplateAccountID, datatyper.UrlTemplateBlodId, datatyper.UrlTemplateType, datatyper.UrlTemplateName)
	uploadPath := fmt.Sprintf("%supload/%s", jh.Path, datatyper.UrlTemplateAccountID)
	eventSourcePath := fmt.Sprintf("%seventsource/?types=%s&closeafter=%s&ping=%s", jh.Path, datatyper.UrlTemplateTypes, datatyper.UrlTemplateClosedAfter, datatyper.UrlTemplatePing)

	sessionHandler := NewSessionHandler(
		NewAccountRepo(),
		jh.sessionCapabilityInfo,
		baseURL+apiPath,
		baseURL+downloadPath,
		baseURL+uploadPath,
		baseURL+eventSourcePath,
		jh.Logger,
	)

	apiHandler := NewAPIHandler(jh.Capability, StubSessionStater{}, jh.contextUserKey, store.OpenAccount, jh.Logger)
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
			if !methodAccepted {
				resp.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			nextHandler.ServeHTTP(resp, req)
		}
	}

	authMW := NewAuthenticationMiddleware(store.OpenEmailAuth, jh.Logger, jh.contextUserKey)

	corsMR := NewCORSMiddleware(jh.CORSAllowFrom, []string{"Authorization", "*"})

	//create a new mux for routing in this path
	mux := http.NewServeMux()
	mux.HandleFunc(sessionPath,
		getRejectUnsupportedMethodsHandler([]string{http.MethodGet, http.MethodOptions},
			corsMR.HandleMethodOptions(
				authMW.Authenticate(sessionHandler))))

	jh.Logger.Debug("register path", slog.Any("sessionPath", sessionPath))
	mux.HandleFunc(apiPath, getRejectUnsupportedMethodsHandler([]string{http.MethodPost, http.MethodOptions},
		corsMR.HandleMethodOptions(
			authMW.Authenticate(apiHandler))))

	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		//do dome pattern matching here
		/*
			go over the remaing candidates and see if we have match
			substitute the templates with a wildcard which matches
		*/

		switch getHandlerForPath(req.URL.Path, downloadPath, uploadPath, eventSourcePath) {
		case handlerTypeDownload:
			getRejectUnsupportedMethodsHandler([]string{http.MethodGet, http.MethodOptions},
				corsMR.HandleMethodOptions(
					authMW.Authenticate(downloadHandler)))
			return
		case handlerTypeUpload:
			getRejectUnsupportedMethodsHandler([]string{http.MethodPost, http.MethodOptions},
				corsMR.HandleMethodOptions(
					authMW.Authenticate(uploadHandler)))
			return
		case handlerTypeEventSource:
			getRejectUnsupportedMethodsHandler([]string{http.MethodGet, http.MethodOptions},
				corsMR.HandleMethodOptions(
					authMW.Authenticate(eventSourceHandler)))
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

// AddCORSAllowedOriginHeader sets a CORS header when a context value indicates we should do so
func AddCORSAllowedOriginHeader(w http.ResponseWriter, r *http.Request) {
	if corsAllowOriging := r.Context().Value(corsAllowOriginCtxKey); corsAllowOriging != nil {
		if corsAllowOrigingStr, ok := corsAllowOriging.(string); ok && corsAllowOrigingStr != "" {
			w.Header().Set("Access-Control-Allow-Origin", corsAllowOrigingStr)
		}
	}
}
