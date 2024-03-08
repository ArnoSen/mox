package mailcapability

import (
	"context"
	"log/slog"

	"github.com/mjl-/bstore"
	"github.com/mjl-/mox/jmapserver/basetypes"
	"github.com/mjl-/mox/jmapserver/capabilitier"
	"github.com/mjl-/mox/jmapserver/jaccount"
	"github.com/mjl-/mox/jmapserver/mlevelerrors"
	"github.com/mjl-/mox/mlog"
	"github.com/mjl-/mox/store"
)

var _ capabilitier.Getter = &ThreadDT{}

type ThreadDT struct {
	mlog mlog.Log
}

func NewThread(mlog mlog.Log) ThreadDT {
	return ThreadDT{
		mlog: mlog,
	}
}

func (t ThreadDT) Name() string {
	return "Thread"
}

func (tDT ThreadDT) Get(ctx context.Context, jaccount jaccount.JAccounter, accountId basetypes.Id, ids []basetypes.Id, properties []string, customParams any) (retAccountId basetypes.Id, state string, list []interface{}, notFound []basetypes.Id, mErr *mlevelerrors.MethodLevelError) {

	var result []any

	for _, id := range ids {
		idInt64, err := id.Int64()
		if err != nil {
			//the email ids are imap ids meaning they are int64
			notFound = append(notFound, id)
			continue
		}

		q := bstore.QueryDB[store.Message](ctx, jaccount.DB())
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
				tDT.mlog.Error("error getting next id", slog.Any("err", err.Error()))
				return accountId, "", nil, nil, mlevelerrors.NewMethodLevelErrorServerFail()
			}
			th.EmailIds = append(th.EmailIds, basetypes.NewIdFromInt64(mailID))
		}
		result = append(result, th)
	}
	if notFound == nil {
		notFound = []basetypes.Id{}
	}

	return accountId, state, list, notFound, mErr
}

// https://datatracker.ietf.org/doc/html/rfc8620#section-5.2
func (tDT ThreadDT) Changes(ctx context.Context, jaccount jaccount.JAccounter, accountId basetypes.Id, sinceState string, maxChanges *basetypes.Uint) (retAccountId basetypes.Id, oldState string, newState string, hasMoreChanges bool, created []basetypes.Id, updated []basetypes.Id, destroyed []basetypes.Id, mErr *mlevelerrors.MethodLevelError) {
	//TODO need to add modseq for threads or find a way to get this behavior from emails
	//AO: not sure what to send back with regards to oldstate/newstate
	mErr = mlevelerrors.NewMethodLevelErrorCannotCalculateChanges()
	return
}

func (tDT ThreadDT) CustomGetRequestParams() any {
	return nil
}

// ../../rfc/8621:1142
type Thread struct {
	Id       basetypes.Id   `json:"id"`
	EmailIds []basetypes.Id `json:"emailIds"`
}
