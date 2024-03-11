package capabilitier

import (
	"github.com/mjl-/bstore"
	"github.com/mjl-/mox/mlog"
	"github.com/mjl-/mox/store"
)

// JAccount is an adaptor for a mox account. It serves the JMAP specific datatypes
// Ideally this package should be removed over time and all logic should be moved to the mox core packages
// that have knowlegde about what properties are stored in persistent storage and what properties are calculated
type JAccounter interface {
	Close() error
	DB() *bstore.DB
	Account() *store.Account
}

type AccountEmailer interface {
}

var _ JAccounter = &JAccount{}

type JAccount struct {
	mAccount *store.Account
	mlog     mlog.Log
}

func NewJAccount(mAccount *store.Account, mlog mlog.Log) *JAccount {
	return &JAccount{
		mAccount: mAccount,
		mlog:     mlog,
	}
}

func (ja JAccount) Close() error {
	return ja.mAccount.Close()
}

func (ja JAccount) DB() *bstore.DB {
	return ja.mAccount.DB
}

func (ja JAccount) Account() *store.Account {
	return ja.mAccount
}
