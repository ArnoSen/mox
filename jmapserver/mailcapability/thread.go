package mailcapability

import (
	"context"

	"github.com/mjl-/mox/jmapserver/basetypes"
	"github.com/mjl-/mox/jmapserver/capabilitier"
	"github.com/mjl-/mox/jmapserver/jaccount"
	"github.com/mjl-/mox/jmapserver/mlevelerrors"
)

var _ capabilitier.Getter = &ThreadDT{}

type ThreadDT struct {
}

func NewThread() ThreadDT {
	return ThreadDT{}
}

func (t ThreadDT) Name() string {
	return "Thread"
}

func (tDT ThreadDT) Get(ctx context.Context, jaccount jaccount.JAccounter, accountId basetypes.Id, ids []basetypes.Id, properties []string, customParams any) (retAccountId basetypes.Id, state string, list []interface{}, notFound []basetypes.Id, mErr *mlevelerrors.MethodLevelError) {

	state, result, notFound, mErr := jaccount.Thread().Get(ctx, ids)
	for _, r := range result {
		list = append(list, r)
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
