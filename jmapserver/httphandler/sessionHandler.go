package httphandler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"log/slog"

	"github.com/mjl-/mox/jmapserver/user"

	"github.com/mjl-/mox/jmapserver/mailcapability"
	"github.com/mjl-/mox/mlog"
)

type AccountRepoer interface {
	//GetAccountsForUser returns the accounts of an user
	GetAccounts(ctx context.Context, userID string) (map[string]Account, error)
	GetPrimaryAccounts(ctx context.Context, userID string) (map[string]string, error)
}

// AccountRepo implements AccountRepoer
type AccountRepo struct{}

func NewAccountRepo() AccountRepo {
	return AccountRepo{}
}

func (ar AccountRepo) GetAccounts(ctx context.Context, userID string) (map[string]Account, error) {
	//TODO this will end up in a DB someday
	return map[string]Account{
		"000": Account{
			Name:        userID,
			IsPersonal:  true,
			IsReadyOnly: false,
			AccountCapabilities: map[string]interface{}{
				mailcapability.URN: mailcapability.NewDefaultMailCapabilitySettings(),
			},
		},
	}, nil

}

func (ar AccountRepo) GetPrimaryAccounts(ctx context.Context, userID string) (map[string]string, error) {
	//FIXME remove static content
	return map[string]string{
		mailcapability.URN: "000",
	}, nil
}

type Session struct {
	Capabilities    map[string]interface{} `json:"capabilities"`
	Accounts        map[string]Account     `json:"accounts"`
	PrimaryAccounts map[string]string      `json:"primaryAccounts"`
	Username        string                 `json:"username"`
	APIURL          string                 `json:"apiUrl"`
	DownloadURL     string                 `json:"downloadUrl"`
	UploadURL       string                 `json:"uploadUrl"`
	EventSourceURL  string                 `json:"eventSourceUrl"`
	State           string                 `json:"state"`
}

type Account struct {
	Name                string                 `json:"name"`
	IsPersonal          bool                   `json:"isPersonal"`
	IsReadyOnly         bool                   `json:"isReadyOnly"`
	AccountCapabilities map[string]interface{} `json:"accountCapabilities"`
}

type SessionHandler struct {
	sessionApp *SessionApp

	//CacheControlHeader contains a optional cache control header
	CacheControlHeader [2]string
	contextUserKey     string
	logger             mlog.Log
}

// baseURL must have format scheme://host:port
func NewSessionHandler(sessionApp *SessionApp, contextUserKey string, logger mlog.Log) SessionHandler {
	result := SessionHandler{
		sessionApp:         sessionApp,
		contextUserKey:     contextUserKey,
		logger:             logger,
		CacheControlHeader: [2]string{"Cache-Control", "no-cache, no-store, must-revalidate"},
	}

	return result

}

func (sh SessionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	ctUserxVal := r.Context().Value(sh.contextUserKey)
	user, ok := ctUserxVal.(user.User)
	if !ok || user.Email == "" {
		if !ok {
			sh.logger.Debug(fmt.Sprintf("ctxUserxVal is not of type user.User but %T", ctUserxVal))
		} else {
			sh.logger.Debug("username is context value of type user.User is empty")
		}
		//user is not authenticated so send error
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	result, err := sh.sessionApp.Session(r.Context(), user.Email)
	if err != nil {
		sh.logger.Error("errr getting session object", slog.Any("err", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
		return
	}

	w.Header().Set(HeaderContentType, HeaderContentTypeJSON)
	if len(sh.CacheControlHeader) == 2 {
		w.Header().Set(sh.CacheControlHeader[0], sh.CacheControlHeader[1])
	}

	AddCORSAllowedOriginHeader(w, r)
	w.Write(resultBytes)
}

// SessionApp contains all variables to generate the session object. It is decoupled from any http handler stuff so it can be passed into the api handler app
type SessionApp struct {
	Host                                           string
	AccountRepo                                    AccountRepoer
	Capabilities                                   map[string]interface{}
	APIURL, DownloadURL, UploadURL, EventSourceURL string

	//stateHashingFunc is the hashs algo used to generate a state value
	stateHashingFunc func([]byte) []byte

	logger mlog.Log
}

// NewSessionApp instantiates a new session app
func NewSessionApp(host string, accountRepo AccountRepoer, capabilities map[string]interface{}, apiURL, downloadURL, uploadURL, eventSourceURL string, hf func([]byte) []byte, logger mlog.Log) *SessionApp {
	return &SessionApp{
		Host:             host,
		AccountRepo:      accountRepo,
		Capabilities:     capabilities,
		APIURL:           apiURL,
		DownloadURL:      downloadURL,
		UploadURL:        uploadURL,
		EventSourceURL:   eventSourceURL,
		stateHashingFunc: hf,
		logger:           logger,
	}
}

// Session returns the session object that is returned when the session object is requested
func (sa SessionApp) Session(ctx context.Context, email string) (*Session, error) {
	//NB: this returns stub information at the moment
	accounts, err := sa.AccountRepo.GetAccounts(ctx, email)
	if err != nil {
		return nil, err
	}

	//NB: this returns stub information at the moment
	primaryAccounts, err := sa.AccountRepo.GetPrimaryAccounts(ctx, email)
	if err != nil {
		return nil, err
	}

	result := &Session{
		//set everything except for state
		Capabilities:    sa.Capabilities,
		Accounts:        accounts,
		PrimaryAccounts: primaryAccounts,
		Username:        email,
		APIURL:          sa.Host + sa.APIURL,
		DownloadURL:     sa.Host + sa.DownloadURL,
		UploadURL:       sa.Host + sa.UploadURL,
		EventSourceURL:  sa.Host + sa.EventSourceURL,
	}

	//calculate a hash of the object that is used for setting a State
	//FIXME maybe for performance it is better to come up with an implementation that doesn't have to marshal things twice
	sessionJSONBytest, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	//take a base64 of the hashing result
	result.State = base64.StdEncoding.EncodeToString(sa.stateHashingFunc(sessionJSONBytest))

	return result, nil
}

// State returns the state of the session object. It implements httphander.SessionStater . This is done to not leak more information about the session object then necessary
func (sa SessionApp) SessionState(ctx context.Context, email string) (string, error) {
	session, err := sa.Session(ctx, email)
	if err != nil {
		return "", nil
	}
	return session.State, nil

}
