package jaccount

import (
	"context"

	"log/slog"

	"github.com/mjl-/bstore"
	"github.com/mjl-/mox/jmapserver/basetypes"
	"github.com/mjl-/mox/jmapserver/mlevelerrors"
	"github.com/mjl-/mox/mlog"
	"github.com/mjl-/mox/store"
)

// ../../rfc/8621:1142
type Thread struct {
	Id       basetypes.Id   `json:"id"`
	EmailIds []basetypes.Id `json:"emailIds"`
}

type AccountThread struct {
	mAccount *store.Account
	mlog     mlog.Log
}

func NewAccountThread(mAccount *store.Account, mlog mlog.Log) *AccountThread {
	return &AccountThread{
		mAccount: mAccount,
		mlog:     mlog,
	}
}

// ../../rfc/8621:1183
func (ja *AccountThread) Get(ctx context.Context, ids []basetypes.Id) (state string, result []Thread, notFound []basetypes.Id, mErr *mlevelerrors.MethodLevelError) {
	for _, id := range ids {
		idInt64, err := id.Int64()
		if err != nil {
			//the email ids are imap ids meaning they are int64
			notFound = append(notFound, id)
			continue
		}

		q := bstore.QueryDB[store.Message](ctx, ja.mAccount.DB)
		q.FilterEqual("ThreadID", idInt64)
		q.FilterEqual("Deleted", false)
		q.FilterEqual("Expunged", false)

		th := Thread{
			Id: id,
		}

		for {
			var mailID int64
			if err := q.NextID(&mailID); err == bstore.ErrAbsent {
				break
			} else if err != nil {
				ja.mlog.Error("error getting next id", slog.Any("err", err.Error()))
				return "", nil, nil, mlevelerrors.NewMethodLevelErrorServerFail()
			}
			th.EmailIds = append(th.EmailIds, basetypes.NewIdFromInt64(mailID))
		}
		result = append(result, th)
	}
	return "stubstate", result, notFound, nil
}
