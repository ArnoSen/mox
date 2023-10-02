package jaccount

import (
	"context"
	"fmt"
	"strconv"

	"github.com/mjl-/bstore"
	"github.com/mjl-/mox/jmapserver/basetypes"
	"github.com/mjl-/mox/jmapserver/mlevelerrors"
	"github.com/mjl-/mox/store"
)

var validEmailFilters []string = []string{
	"inMailbox",
	"inMailboxOtherThan",
	"before",
	"after",
	"minSize",
	"maxSize",
	"allInThreadHaveKeyword",
	"someInThreadHaveKeyword",
	"noneInThreadHaveKeyword",
	"hasKeyword",
	"notKeyword",
	"hasAttachment",
	"text",
	"from",
	"to",
	"cc",
	"bcc",
	"subject",
	"body",
	"header",
}

var validSortProperties []string = []string{
	"receivedAt",
	"size",
	"from",
	"to",
	"subject",
	"sentAt",
	"hasKeyword",
	"allInThreadHaveKeyword",
	"someInThreadHaveKeyword",
}

func (ja *JAccount) QueryEmail(ctx context.Context, filter *basetypes.Filter, sort []basetypes.Comparator, position basetypes.Int, anchor *basetypes.Id, anchorOffset basetypes.Int, limit int, calculateTotal bool) (queryState string, canCalculateChanges bool, retPosition basetypes.Int, ids []basetypes.Id, total basetypes.Uint, mErr *mlevelerrors.MethodLevelError) {

	//FIXME this implementation needs to be completed

	q := bstore.QueryDB[store.Message](ctx, ja.mAccount.DB)

	if filter != nil {
		filterCondition, ok := filter.GetFilter().(basetypes.FilterCondition)
		if !ok {
			//let's do only simple filters for now
			return "", false, 0, nil, 0, mlevelerrors.NewMethodLevelErrorUnsupportedFilter("only filterconditions are supported for now")
		}

		switch filterCondition.Property {
		case "inMailbox":
			var mailboxIDint int64
			switch filterCondition.AssertedValue.(type) {
			case int:
				mailboxIDint = int64(filterCondition.AssertedValue.(int))
			case string:
				var parseErr error
				mailboxIDint, parseErr = strconv.ParseInt(filterCondition.AssertedValue.(string), 10, 64)
				if parseErr != nil {
					return "", false, 0, nil, 0, mlevelerrors.NewMethodLevelErrorUnsupportedFilter("inMailbox filter value must be a (quoted) integer")
				}
			default:
				return "", false, 0, nil, 0, mlevelerrors.NewMethodLevelErrorUnsupportedFilter("inMailbox filter value must be a (quoted) integer")
			}

			q.FilterNonzero(store.Message{
				MailboxID: int64(mailboxIDint),
			})
		default:
			return "", false, 0, nil, 0, mlevelerrors.NewMethodLevelErrorUnsupportedFilter("unsupported filter")
		}

	}

	for _, s := range sort {
		switch s.Property {
		case "receivedAt":
			if s.IsAscending {
				q.SortAsc("Received")
			}
			q.SortDesc("Received")
		default:
			return "", false, 0, nil, 0, mlevelerrors.NewMethodLevelErrorUnsupportedSort("unsupported sort parameter")
		}
	}

	q.Limit(limit)

	if calculateTotal {
		//TODO looking at the implementation of Count, maybe it is better we calc the total in the next for loop
		totalCnt, countErr := q.Count()
		if countErr != nil {
			return "", false, 0, nil, 0, mlevelerrors.NewMethodLevelErrorServerFail()
		}
		total = basetypes.Uint(totalCnt)
	}

	var (
		//FIXME position can also be negative. In that case results need to come from the other end of the list.
		skip int64 = int64(position)
		i    int64
	)

	for {
		i++
		if i-1 < skip {
			continue
		}

		var id uint64
		if err := q.NextID(&id); err == bstore.ErrAbsent {
			// No more messages.
			// Note: if we don't iterate until an error, Close must be called on the query for cleanup.
			break
		} else if err != nil {
			return "", false, 0, nil, 0, mlevelerrors.NewMethodLevelErrorServerFail()
		}
		// The ID is fetched from the index. The full record is
		// never read from the database. Calling Next instead
		// of NextID does always fetch, parse and return the
		// full record.
		ids = append(ids, basetypes.Id(fmt.Sprintf("%s", id)))
	}

	return "stubstate", false, position, ids, total, nil
}
