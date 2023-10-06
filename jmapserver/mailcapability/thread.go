package mailcapability

import (
	"context"

	"github.com/mjl-/mox/jmapserver/basetypes"
	"github.com/mjl-/mox/jmapserver/jaccount"
	"github.com/mjl-/mox/jmapserver/mlevelerrors"
)

type ThreadDT struct {
}

func NewThread() ThreadDT {
	return ThreadDT{}
}

func (t ThreadDT) Name() string {
	return "Thread"
}

func (tDT ThreadDT) Get(ctx context.Context, jaccount jaccount.JAccounter, accountId basetypes.Id, ids []basetypes.Id, properties []string) (retAccountId basetypes.Id, state string, list []interface{}, notFound []basetypes.Id, mErr *mlevelerrors.MethodLevelError) {

	state, result, notFound, mErr := jaccount.GetThread(ctx, ids)
	for _, r := range result {
		list = append(list, r)
	}

	if notFound == nil {
		notFound = []basetypes.Id{}
	}

	return accountId, state, list, notFound, mErr

}
