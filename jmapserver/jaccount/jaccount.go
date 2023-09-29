package jaccount

import (
	"context"

	"github.com/mjl-/mox/jmapserver/basetypes"
	"github.com/mjl-/mox/jmapserver/mlevelerrors"
	"github.com/mjl-/mox/store"
)

// JAccount is an adaptor for a mox account. It serves the JMAP specific datatypes
// Ideally this package should be removed over time and all logic should be moved to the mox core packages
// that have knowlegde about what properties are stored in persistent storage and what properties are calculated
type JAccounter interface {
	GetMailboxes(ctx context.Context, ids []basetypes.Id) ([]Mailbox, []basetypes.Id, string, *mlevelerrors.MethodLevelError)

	QueryEmail(ctx context.Context, filter *basetypes.Filter, sort []basetypes.Comparator, position basetypes.Int, anchor *basetypes.Id, anchorOffset basetypes.Int, limit int, calculateTotal bool) (queryState string, canCalculateChanges bool, retPosition basetypes.Int, ids []basetypes.Id, total basetypes.Uint, mErr *mlevelerrors.MethodLevelError)
}

var _ JAccounter = &JAccount{}

type JAccount struct {
	mAccount *store.Account
}

func NewJAccount(mAccount *store.Account) *JAccount {
	return &JAccount{
		mAccount: mAccount,
	}
}
