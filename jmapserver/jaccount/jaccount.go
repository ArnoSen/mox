package jaccount

import (
	"context"
	"fmt"

	"github.com/mjl-/bstore"
	"github.com/mjl-/mox/jmapserver/basetypes"
	"github.com/mjl-/mox/jmapserver/mlevelerrors"
	"github.com/mjl-/mox/store"
)

// JAccount is an adaptor for a mox account. It serves the JMAP specific datatypes
type JAccounter interface {
	GetMailboxes(ctx context.Context, ids []basetypes.Id) ([]Mailbox, []basetypes.Id, string, *mlevelerrors.MethodLevelError)
}

type JAccount struct {
	mAccount *store.Account
}

func NewJAccount(mAccount *store.Account) *JAccount {
	return &JAccount{mAccount}
}

func (ja *JAccount) GetMailboxes(ctx context.Context, ids []basetypes.Id) (result []Mailbox, notFound []basetypes.Id, state string, mErr *mlevelerrors.MethodLevelError) {

	//FIXME need to implement selection of specific mailboxes

	q := bstore.QueryDB[store.Mailbox](ctx, ja.mAccount.DB)

	mbs, err := q.List()
	if err != nil {
		mErr = mlevelerrors.NewMethodLevelErrorServerFail()
		return
	}

	for _, mb := range mbs {
		result = append(result, Mailbox{
			Id:   basetypes.Id(fmt.Sprintf("%d", mb.ID)),
			Name: mb.Name,

			ParentId:      nil,
			Role:          "",
			SortOrder:     0,
			TotalEmails:   0,
			UnreadEmails:  0,
			TotalThreads:  0,
			UnreadThreads: 0,
			MyRights: MailboxRights{
				MayReadItems:   true,
				MayAddItems:    true,
				MayRemoveItems: true,
				MaySetSeen:     true,
				MaySetKeywords: true,
				MayCreateChild: true,
				MayRename:      true,
				MayDelete:      true,
				MaySubmit:      true,
			},
			IsSubscribed: false,
		})
	}
	return
}
