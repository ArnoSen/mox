package jaccount

import (
	"context"

	"github.com/mjl-/bstore"
	"github.com/mjl-/mox/jmapserver/basetypes"
	"github.com/mjl-/mox/jmapserver/mlevelerrors"
	"github.com/mjl-/mox/mlog"
	"github.com/mjl-/mox/store"
)

type Thread struct {
	Id       basetypes.Id   `json:"id"`
	EmailIds []basetypes.Id `json:"emailIds"`
}

func (ja *JAccount) GetThread(ctx context.Context, ids []basetypes.Id) (state string, result []Thread, notFound []basetypes.Id, mErr *mlevelerrors.MethodLevelError) {
	for _, id := range ids {
		idInt64, err := id.Int64()
		if err != nil {
			//the email ids are imap ids meaning they are int64
			notFound = append(notFound, id)
			continue
		}

		q := bstore.QueryDB[store.Message](ctx, ja.mAccount.DB)
		q.FilterEqual("ThreadID", idInt64)

		th := Thread{
			Id: id,
		}

		for {
			var mailID int64
			if err := q.NextID(&mailID); err == bstore.ErrAbsent {
				break
			} else if err != nil {
				ja.mlog.Error("error getting next id", mlog.Field("err", err.Error()))
				return "", nil, nil, mlevelerrors.NewMethodLevelErrorServerFail()
			}
			th.EmailIds = append(th.EmailIds, basetypes.NewIdFromInt64(mailID))
		}
		result = append(result, th)
	}
	return "stubstate", result, notFound, nil
}
