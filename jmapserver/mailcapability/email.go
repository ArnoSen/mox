package mailcapability

import (
	"context"

	"github.com/mjl-/mox/jmapserver/basetypes"
	"github.com/mjl-/mox/jmapserver/jaccount"
	"github.com/mjl-/mox/jmapserver/mlevelerrors"
)

type EmailDT struct {
	//maxQueryLimit is the number of emails returned for a query request
	maxQueryLimit int
}

func NewEmail(maxQueryLimit int) EmailDT {
	return EmailDT{
		maxQueryLimit: maxQueryLimit,
	}
}

func (m EmailDT) Name() string {
	return "Email"
}

// https://datatracker.ietf.org/doc/html/rfc8620#section-5.5
func (m EmailDT) Query(ctx context.Context, jaccount jaccount.JAccounter, accountId basetypes.Id, filter *basetypes.Filter, sort []basetypes.Comparator, position basetypes.Int, anchor *basetypes.Id, anchorOffset basetypes.Int, limit *basetypes.Uint, calculateTotal bool) (retAccountId basetypes.Id, queryState string, canCalculateChanges bool, retPosition basetypes.Int, ids []basetypes.Id, total basetypes.Uint, retLimit basetypes.Uint, mErr *mlevelerrors.MethodLevelError) {

	//FIXME
	//Need to handle collapseThreads ../../rfc/8621:2506

	var adjustedLimit int = m.maxQueryLimit

	if limit != nil && int(*limit) < adjustedLimit {
		adjustedLimit = int(*limit)
	}

	state, canCalculateChanges, retPosition, ids, total, mErr := jaccount.QueryEmail(ctx, filter, sort, position, anchor, anchorOffset, adjustedLimit, calculateTotal)

	if ids == nil {
		//send an empty array instead of a null value to not break the current way of resolving request references
		ids = []basetypes.Id{}
	}

	return accountId, state, canCalculateChanges, basetypes.Int(retPosition), ids, total, basetypes.Uint(adjustedLimit), mErr
}

// https://datatracker.ietf.org/doc/html/rfc8620#section-5.1
func (m EmailDT) Get(ctx context.Context, jaccount jaccount.JAccounter, accountId basetypes.Id, ids []basetypes.Id, properties []string) (retAccountId basetypes.Id, state string, list []interface{}, notFound []basetypes.Id, mErr *mlevelerrors.MethodLevelError) {

	//property filtering is done at the handler level. It is included here so we can check if some fields are needed in the result
	state, result, notFound, mErr := jaccount.GetEmail(ctx, ids, properties)

	for _, r := range result {
		list = append(list, r)
	}

	if list == nil {
		//always return an empty slice
		list = []interface{}{}
	}

	if notFound == nil {
		//send an empty array instead of a null value to not break the current way of resolving request references
		notFound = []basetypes.Id{}
	}

	return accountId, state, list, notFound, mErr

}
