package jaccount

import (
	"context"

	"github.com/mjl-/mox/jmapserver/basetypes"
	"github.com/mjl-/mox/jmapserver/mlevelerrors"
	"github.com/mjl-/mox/mlog"
	"github.com/mjl-/mox/store"
)

// JAccount is an adaptor for a mox account. It serves the JMAP specific datatypes
// Ideally this package should be removed over time and all logic should be moved to the mox core packages
// that have knowlegde about what properties are stored in persistent storage and what properties are calculated
type JAccounter interface {
	Mailbox() AccountMailboxer
	Email() AccountEmailer
	Thread() AccountThreader
	Close() error
}

type AccountMailboxer interface {
	Get(ctx context.Context, ids []basetypes.Id) ([]Mailbox, []basetypes.Id, string, *mlevelerrors.MethodLevelError)
}

type AccountEmailer interface {
	Get(ctx context.Context, ids []basetypes.Id, properties, bodyProperties []string, FetchTextBodyValues, FetchHTMLBodyValues, FetchAllBodyValues bool, MaxBodyValueBytes *basetypes.Uint) (result []Email, notFound []basetypes.Id, mErr *mlevelerrors.MethodLevelError)
	Query(ctx context.Context, filter *basetypes.Filter, sort []basetypes.Comparator, position basetypes.Int, anchor *basetypes.Id, anchorOffset basetypes.Int, limit int, calculateTotal bool, collapseThreads bool) (queryState string, canCalculateChanges bool, retPosition basetypes.Int, ids []basetypes.Id, total basetypes.Uint, mErr *mlevelerrors.MethodLevelError)
	Set(ctx context.Context, ifInState *string, create map[basetypes.Id]interface{}, update map[basetypes.Id]basetypes.PatchObject, destroy []basetypes.Id) (oldState *string, newState string, created map[basetypes.Id]interface{}, updated map[basetypes.Id]interface{}, destroyed map[basetypes.Id]interface{}, notCreated map[basetypes.Id]mlevelerrors.SetError, notUpdated map[basetypes.Id]mlevelerrors.SetError, notDestroyed map[basetypes.Id]mlevelerrors.SetError, mErr *mlevelerrors.MethodLevelError)
	Changes(ctx context.Context, accountId basetypes.Id, sinceState string, maxChanges *basetypes.Uint) (retAccountId basetypes.Id, oldState string, newState string, hasMoreChanges bool, created, updated, destroyed []basetypes.Id, mErr *mlevelerrors.MethodLevelError)
	State(ctx context.Context) (string, *mlevelerrors.MethodLevelError)
	DownloadBlob(ctx context.Context, blobID, name, Type string) (bool, []byte, error)
}

type AccountThreader interface {
	Get(ctx context.Context, ids []basetypes.Id) (state string, result []Thread, notFound []basetypes.Id, mErr *mlevelerrors.MethodLevelError)
}

var _ JAccounter = &JAccount{}

type JAccount struct {
	mAccount       *store.Account
	mlog           mlog.Log
	AccountEmail   AccountEmailer
	AccountMailbox AccountMailboxer
	AccountThread  AccountThreader
}

func NewJAccount(mAccount *store.Account, mlog mlog.Log) *JAccount {
	return &JAccount{
		mAccount:       mAccount,
		mlog:           mlog,
		AccountEmail:   NewAccountEmail(mAccount, mlog),
		AccountMailbox: NewAccountMailbox(mAccount, mlog),
		AccountThread:  NewAccountThread(mAccount, mlog),
	}
}

func (ja *JAccount) Email() AccountEmailer {
	return ja.AccountEmail
}

func (ja *JAccount) Mailbox() AccountMailboxer {
	return ja.AccountMailbox
}

func (ja *JAccount) Thread() AccountThreader {
	return ja.AccountThread
}

func (ja JAccount) Close() error {
	return ja.mAccount.Close()
}

// DownloadBlob returns the raw contents of a blobid. The first param in the reponse indicates if the blob was found
func (ja JAccount) DownloadBlob(ctx context.Context, blobID, name, Type string) (bool, []byte, error) {
	//TODO download is in the global namespace so we should determine here to what capability this request should go to
	// for now only email is supported
	return ja.Email().DownloadBlob(ctx, blobID, name, Type)
}
