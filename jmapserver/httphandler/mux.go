package httphandler

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/mjl-/mox/jmapserver/capabilitier"
	"github.com/mjl-/mox/jmapserver/core"
	"github.com/mjl-/mox/jmapserver/mailcapability"
	"github.com/mjl-/mox/jmapserver/user"
	"github.com/mjl-/mox/mlog"
	"github.com/mjl-/mox/store"
)

const (
	corsAllowOriginCtxKey = "Access-Control-Allow-Origin"
	corsAllowOriginHeader = "Access-Control-Allow-Origin"
	defaultContextUserKey = "_user"

	sessionRoute     = "session"
	apiRoute         = "api"
	downloadRoute    = "download/"
	uploadRoute      = "upload/"
	eventsourceRoute = "eventsource/"

	//used in download  and upload
	UrlTemplateAccountID = "{accountId}"

	//used in download
	UrlTemplateBlodId = "{blobId}"
	UrlTemplateName   = "{name}"
	UrlTemplateType   = "{type}"

	//used in Eventsource path
	UrlTemplateTypes       = "{types}"
	UrlTemplateClosedAfter = "{closeafter}"
	UrlTemplatePing        = "{ping}"
)

type JMAPServerHandler struct {
	//Path is the absolute path of the jmap handler as set in http/web.go
	Path string

	//Hostname is the hostname that we are listening on. This is send in the session handler
	Hostname string

	//Port is the port that we are listening on. This is send in the session handler
	Port int

	OpenEmailAuthFunc OpenEmailAuthFunc
	AccountOpener     AccountOpener

	//CORSAllowFrom defines the hosts that can access JMAP resources from a browser
	CORSAllowFrom []string
	Logger        mlog.Log

	contextUserKey string

	sessionPath, apiPath, downloadPath, uploadPath, eventsourcePath string

	sessionHandler, apiHandler, downloadHandler, uploadHandler, eventSourceHandler http.Handler
}

func NewHandler(hostname, path string, port int, openEmailAuthFunc OpenEmailAuthFunc, accountOpener AccountOpener, corsAllowFrom []string, logger mlog.Log) JMAPServerHandler {

	capability := []capabilitier.Capabilitier{
		core.NewCore(core.CoreCapabilitySettings{
			// ../../rfc/8620:517
			//use the minimum recommended values for now. Maybe move some to settings later on
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
	}

	sessionCapabilityInfo := make(map[string]interface{})
	for _, capability := range capability {
		sessionCapabilityInfo[capability.Urn()] = capability.SessionObjectInfo()
	}

	downloadPath := fmt.Sprintf("%s%s%s/%s/%s?type=%s", path, downloadRoute, UrlTemplateAccountID, UrlTemplateBlodId, UrlTemplateName, UrlTemplateType)
	uploadPath := fmt.Sprintf("%s%s%s", path, uploadRoute, UrlTemplateAccountID)
	eventSourcePath := fmt.Sprintf("%s%s?types=%s&closeafter=%s&ping=%s", path, eventsourceRoute, UrlTemplateTypes, UrlTemplateClosedAfter, UrlTemplatePing)

	result := JMAPServerHandler{
		Hostname:          hostname,
		Port:              port,
		Path:              path,
		CORSAllowFrom:     corsAllowFrom,
		OpenEmailAuthFunc: openEmailAuthFunc,
		AccountOpener:     accountOpener,
		Logger:            logger,
		contextUserKey:    defaultContextUserKey,
		// ../../rfc/8620:679
		sessionHandler: NewSessionHandler(
			fmt.Sprintf("https://%s:%d", hostname, port),
			NewAccountRepo(),
			sessionCapabilityInfo,
			path+"api",
			downloadPath,
			uploadPath,
			eventSourcePath,
			logger,
		),
		apiHandler:         NewAPIHandler(capability, StubSessionStater{}, defaultContextUserKey, store.OpenAccount, logger),
		downloadHandler:    NewDownloadHandler(store.OpenAccount, defaultContextUserKey, downloadPath, logger),
		uploadHandler:      NewUploadHandler(logger),
		eventSourceHandler: NewEventSourceHandler(logger),

		sessionPath:     path + sessionRoute,
		apiPath:         path + apiRoute,
		downloadPath:    downloadPath,
		uploadPath:      uploadPath,
		eventsourcePath: eventSourcePath,
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
	mux.HandleFunc(jh.sessionPath, getRejectUnsupportedMethodsHandler([]string{http.MethodGet, http.MethodOptions},
		corsMR.HandleMethodOptions(
			authMW.Authenticate(jh.sessionHandler))))

	mux.HandleFunc(jh.apiPath, getRejectUnsupportedMethodsHandler([]string{http.MethodPost, http.MethodOptions},
		corsMR.HandleMethodOptions(
			authMW.Authenticate(jh.apiHandler))))

	mux.HandleFunc(jh.Path+downloadRoute, getRejectUnsupportedMethodsHandler([]string{http.MethodGet, http.MethodOptions},
		corsMR.HandleMethodOptions(
			authMW.Authenticate(jh.downloadHandler))))

	mux.HandleFunc(jh.Path+uploadRoute, getRejectUnsupportedMethodsHandler([]string{http.MethodPost, http.MethodOptions},
		corsMR.HandleMethodOptions(
			authMW.Authenticate(jh.uploadHandler))))

	mux.HandleFunc(jh.Path+eventsourceRoute, getRejectUnsupportedMethodsHandler([]string{http.MethodGet, http.MethodOptions},
		corsMR.HandleMethodOptions(jh.eventSourceHandler.ServeHTTP)))

	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		//if nothing matches, we exit here
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("page not found"))
	})

	mux.ServeHTTP(rw, r)
}

// AddCORSAllowedOriginHeader sets a CORS header when a context value indicates we should do so
func AddCORSAllowedOriginHeader(w http.ResponseWriter, r *http.Request) {
	if corsAllowOriging := r.Context().Value(corsAllowOriginCtxKey); corsAllowOriging != nil {
		if corsAllowOrigingStr, ok := corsAllowOriging.(string); ok && corsAllowOrigingStr != "" {
			w.Header().Set("Access-Control-Allow-Origin", corsAllowOrigingStr)
		}
	}
}
