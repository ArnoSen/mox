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
	//Mailbox
	GetMailboxes(ctx context.Context, ids []basetypes.Id) ([]Mailbox, []basetypes.Id, string, *mlevelerrors.MethodLevelError)

	//Email
	GetEmail(ctx context.Context, ids []basetypes.Id, properties, bodyProperties []string, FetchTextBodyValues, FetchHTMLBodyValues, FetchAllBodyValues bool, MaxBodyValueBytes *basetypes.Uint) (state string, result []Email, notFound []basetypes.Id, mErr *mlevelerrors.MethodLevelError)
	QueryEmail(ctx context.Context, filter *basetypes.Filter, sort []basetypes.Comparator, position basetypes.Int, anchor *basetypes.Id, anchorOffset basetypes.Int, limit int, calculateTotal bool, collapseThreads bool) (queryState string, canCalculateChanges bool, retPosition basetypes.Int, ids []basetypes.Id, total basetypes.Uint, mErr *mlevelerrors.MethodLevelError)

	//Thread
	GetThread(ctx context.Context, ids []basetypes.Id) (state string, result []Thread, notFound []basetypes.Id, mErr *mlevelerrors.MethodLevelError)
}

var _ JAccounter = &JAccount{}

type JAccount struct {
	mAccount *store.Account
	mlog     *mlog.Log
}

func NewJAccount(mAccount *store.Account, mlog *mlog.Log) *JAccount {
	return &JAccount{
		mAccount: mAccount,
		mlog:     mlog,
	}
}

func (ja JAccount) NewEmail(em store.Message) (JEmail, *mlevelerrors.MethodLevelError) {
	part, err := em.LoadPart(ja.mAccount.MessageReader(em))
	if err != nil {
		ja.mlog.Error("error loading part", mlog.Field("err", err.Error()))
		return JEmail{}, mlevelerrors.NewMethodLevelErrorServerFail()
	}
	return NewJEmail(em, part, ja.mlog), nil
}
