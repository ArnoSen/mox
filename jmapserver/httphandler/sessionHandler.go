package httphandler

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os/user"

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
	AccountRepo                                    AccountRepoer
	Capabilities                                   map[string]interface{}
	APIURL, DownloadURL, UploadURL, EventSourceURL string

	//CacheControlHeader contains a optional cache control header
	CacheControlHeader [2]string

	//stateHashingFunc is the hashs algo used to generate a state value
	stateHashingFunc func([]byte) []byte

	contextUserKey string

	logger *mlog.Log
}

func NewSessionHandler(accountRepo AccountRepoer, capabilities map[string]interface{}, apiURL, downloadURL, uploadURL, eventSourceURL string, logger *mlog.Log) SessionHandler {
	return SessionHandler{
		AccountRepo:    accountRepo,
		Capabilities:   capabilities,
		APIURL:         apiURL,
		DownloadURL:    downloadURL,
		UploadURL:      uploadURL,
		EventSourceURL: eventSourceURL,
		stateHashingFunc: func(b []byte) []byte {
			md5sum := md5.Sum(b)
			return md5sum[:]
		},
		contextUserKey: defaultContextUserKey,
		logger:         logger,
	}
}

func (sh SessionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	ctUserxVal := r.Context().Value(sh.contextUserKey)
	user, ok := ctUserxVal.(user.User)
	if !ok || user.Username == "" {
		//user is not authenticated so send error
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	accounts, err := sh.AccountRepo.GetAccounts(r.Context(), user.Username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	primaryAccounts, err := sh.AccountRepo.GetPrimaryAccounts(r.Context(), user.Username)
	if err != nil {
		//FIXME send out a body with some more information?
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	result := Session{
		//set everything except for state
		Capabilities:    sh.Capabilities,
		Accounts:        accounts,
		PrimaryAccounts: primaryAccounts,
		Username:        user.Username,
		APIURL:          sh.APIURL,
		DownloadURL:     sh.DownloadURL,
		UploadURL:       sh.UploadURL,
		EventSourceURL:  sh.EventSourceURL,
	}

	//calculate a hash of the object that is used for setting a State
	//FIXME maybe for performance it is better to come up with an implementation that doesn't have to marshal things twice
	sessionJSONBytest, err := json.Marshal(result)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//take a base64 of the hashing result
	result.State = base64.StdEncoding.EncodeToString(sh.stateHashingFunc(sessionJSONBytest))

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

	addCORSAllowedOriginHeader(w, r)
	w.Write(resultBytes)

	/*
		if err := json.NewEncoder(w).Encode(result); err != nil {
			//FIXME will this work or will data already be out and we cannot set an heaeder anymore
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	*/
}
