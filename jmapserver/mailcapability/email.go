package mailcapability

import (
	"context"
	"io"
	"log/slog"
	"strconv"
	"strings"

	"github.com/mjl-/bstore"
	"github.com/mjl-/mox/jmapserver/basetypes"
	"github.com/mjl-/mox/jmapserver/capabilitier"
	"github.com/mjl-/mox/jmapserver/mlevelerrors"
	"github.com/mjl-/mox/mlog"
	"github.com/mjl-/mox/store"
)

type EmailDT struct {
	//maxQueryLimit is the number of emails returned for a query request
	maxQueryLimit int
	mlog          mlog.Log
}

func NewEmailDT(maxQueryLimit int, logger mlog.Log) EmailDT {
	return EmailDT{
		maxQueryLimit: maxQueryLimit,
		mlog:          logger,
	}
}

func (m EmailDT) Name() string {
	return "Email"
}

// https://datatracker.ietf.org/doc/html/rfc8620#section-5.5
func (m EmailDT) Query(ctx context.Context, jaccount capabilitier.JAccounter, accountId basetypes.Id, filter *basetypes.Filter, sort []basetypes.Comparator, position basetypes.Int, anchor *basetypes.Id, anchorOffset basetypes.Int, limit *basetypes.Uint, calculateTotal bool, customParams any) (retAccountId basetypes.Id, queryState string, canCalculateChanges bool, retPosition basetypes.Int, ids []basetypes.Id, total basetypes.Uint, retLimit basetypes.Uint, mErr *mlevelerrors.MethodLevelError) {

	var adjustedLimit = m.maxQueryLimit

	if limit != nil && int(*limit) < adjustedLimit {
		adjustedLimit = int(*limit)
	}

	cust := customParams.(*CustomQueryRequestParams)

	m.mlog.Debug("JAccount QueryEmail", slog.Any("collapseThreads", cust.CollapseThreads))

	q := bstore.QueryDB[store.Message](ctx, jaccount.DB())

	parseMailboxID := func(mID any) (int64, *mlevelerrors.MethodLevelError) {
		switch v := mID.(type) {
		case string:
			var parseErr error
			mailboxIDint, parseErr := strconv.ParseInt(v, 10, 64)
			if parseErr != nil {
				return 0, mlevelerrors.NewMethodLevelErrorUnsupportedFilter("inMailbox filter value must be a (quoted) integer")
			}
			return mailboxIDint, nil
		default:
			return 0, mlevelerrors.NewMethodLevelErrorUnsupportedFilter("inMailbox filter value must be a (quoted) integer")
		}
	}

	if filter != nil {
		// ../../rfc/8621:2527
		switch v := filter.GetFilter().(type) {
		case basetypes.FilterCondition:
			//let's do only simple filters for now

			switch v.Property {
			case "inMailbox":
				mailboxIDint, mErr := parseMailboxID(v.AssertedValue)
				if mErr != nil {
					return accountId, "", false, 0, nil, 0, basetypes.Uint(adjustedLimit), mErr
				}

				q.FilterNonzero(store.Message{
					MailboxID: int64(mailboxIDint),
				})

			default:
				return accountId, "", false, 0, nil, 0, basetypes.Uint(adjustedLimit), mlevelerrors.NewMethodLevelErrorUnsupportedFilter("unsupported filter")
			}
		case basetypes.FilterOperator:
			//FIXME
			//To advance with Mailtemi, the filter condition {\"conditions\":[{\"inMailbox\":\"1\"},{\"inMailbox\":\"2\"},{\"inMailbox\":\"3\"},{\"inMailbox\":\"4\"},{\"inMailbox\":\"5\"},{\"inMailbox\":\"6\"}],\"operator\":\"OR\"} is supported through various shortcuts. When there is another type of filter, this needs to be implemented

			//check if this is a simpel filter that we can directly map to a bstore query
			var (
				hasSubFilterOperator bool
				propertyMap          = map[string]interface{}{}
				singleProp           string
			)
			for _, condition := range v.Conditions {
				if c, ok := condition.(basetypes.FilterCondition); !ok {
					hasSubFilterOperator = true
					propertyMap[c.Property] = nil
					singleProp = c.Property
				}
			}

			if hasSubFilterOperator && len(propertyMap) != 1 {
				return accountId, "", false, 0, nil, 0, basetypes.Uint(adjustedLimit), mlevelerrors.NewMethodLevelErrorUnsupportedFilter("unsupported filter")
			}

			//so we have a simple filter
			switch v.Operator {
			case basetypes.FilterOperatorTypeOR:
				switch singleProp {
				case "inMailbox":
					var values []int64
					for _, cond := range v.Conditions {
						mID, mErr := parseMailboxID(cond.(basetypes.FilterCondition).AssertedValue)
						if mErr != nil {
							return accountId, "", false, 0, nil, 0, basetypes.Uint(adjustedLimit), mlevelerrors.NewMethodLevelErrorUnsupportedFilter("unsupported filter")
						}
						values = append(values, mID)
					}
					q.FilterEqual(singleProp, values)
				}
			default:
				return accountId, "", false, 0, nil, 0, basetypes.Uint(adjustedLimit), mlevelerrors.NewMethodLevelErrorUnsupportedFilter("unsupported filter")
			}

		default:
			return accountId, "", false, 0, nil, 0, basetypes.Uint(adjustedLimit), mlevelerrors.NewMethodLevelErrorUnsupportedFilter("only filterconditions are supported for now")
		}
	}

	for _, s := range sort {
		//FIXME we only support sorting at max one level
		//../../rfc/8621:2708
		switch s.Property {
		case "receivedAt":
			if s.IsAscending {
				q.SortAsc("Received")
			}
			q.SortDesc("Received")
		default:
			return accountId, "", false, 0, nil, 0, basetypes.Uint(adjustedLimit), mlevelerrors.NewMethodLevelErrorUnsupportedSort("unsupported sort parameter")
		}
	}

	q.Limit(adjustedLimit + int(position))

	q.FilterEqual("Deleted", false)
	q.FilterEqual("Expunged", false)

	var (
		//FIXME position can also be negative. In that case results need to come from the other end of the list.
		currentPos int64
		threadMap  map[int64]interface{} = make(map[int64]interface{})
	)

search:
	for {
		if !cust.CollapseThreads {
			var id int64
			if err := q.NextID(&id); err == bstore.ErrAbsent {
				// No more messages.
				// Note: if we don't iterate until an error, Close must be called on the query for cleanup.
				break search
			} else if err != nil {
				m.mlog.Error("error getting next id", slog.Any("err", err.Error()))
				return accountId, "", false, 0, nil, 0, basetypes.Uint(adjustedLimit), mlevelerrors.NewMethodLevelErrorServerFail()
			}

			// The ID is fetched from the index. The full record is
			// never read from the database. Calling Next instead
			// of NextID does always fetch, parse and return the
			// full record.
			if currentPos < int64(position) {
				continue search
			}
			currentPos++

			if len(ids) < adjustedLimit {
				ids = append(ids, basetypes.NewIdFromInt64(id))
			}
			total++
		} else {
			//../../rfc/8621:2785
			msg, err := q.Next()
			if err == bstore.ErrAbsent {
				break search
			} else if err != nil {
				m.mlog.Error("error getting message", slog.Any("err", err.Error()))
				return accountId, "", false, 0, nil, 0, basetypes.Uint(adjustedLimit), mlevelerrors.NewMethodLevelErrorServerFail()
			}

			if _, ok := threadMap[msg.ThreadID]; !ok {

				if currentPos < int64(position) {
					continue search
				}

				if len(ids) < adjustedLimit {
					ids = append(ids, basetypes.NewIdFromInt64(msg.ID))
				}
				threadMap[msg.ThreadID] = nil
				total++
			}
			currentPos++
		}
	}

	var highestModSeq int64

	//we need the highest modseq of the ids as value for state of this query
	for _, id := range ids {
		idInt, err := id.Int64()
		if err != nil {
			//should not happen
			continue
		}
		em := store.Message{
			ID: idInt,
		}

		if err := jaccount.DB().Get(ctx, &em); err == nil {
			if highestModSeq == 0 || em.ModSeq.Client() > highestModSeq {
				highestModSeq = em.ModSeq.Client()
			}
		}
	}
	return accountId, strconv.FormatInt(highestModSeq, 10), false, position, ids, total, basetypes.Uint(adjustedLimit), nil
}

type CustomQueryRequestParams struct {
	CollapseThreads bool `json:"collapseThreads"`
}

func (m EmailDT) CustomQueryRequestParams() any {
	return &CustomQueryRequestParams{}
}

type CustomGetRequestParams struct {
	BodyProperties      []string        `json:"bodyProperties"`
	FetchTextBodyValues bool            `json:"fetchTextBodyValues"`
	FetchHTMLBodyValues bool            `json:"fetchHTMLBodyValues"`
	FetchAllBodyValues  bool            `json:"fetchAllBodyValues"`
	MaxBodyValueBytes   *basetypes.Uint `json:"maxBodyValueBytes"`
}

func (m EmailDT) CustomGetRequestParams() any {
	return &CustomGetRequestParams{}
}

// https://datatracker.ietf.org/doc/html/rfc8620#section-5.1
func (m EmailDT) Get(ctx context.Context, jaccount capabilitier.JAccounter, accountId basetypes.Id, ids []basetypes.Id, properties []string, customParams any) (retAccountId basetypes.Id, state string, list []any, notFound []basetypes.Id, mErr *mlevelerrors.MethodLevelError) {

	cust := customParams.(*CustomGetRequestParams)

	//property filtering is done at the handler level. It is included here so we can check if some fields are needed in the result
	/*
		result, notFound, mErr := capabilitier.Email().Get(ctx, ids, properties, cust.BodyProperties, cust.FetchTextBodyValues, cust.FetchHTMLBodyValues, cust.FetchAllBodyValues, cust.MaxBodyValueBytes)
	*/

	m.mlog.Debug("custom get params", slog.Any("bodyProperties", strings.Join(cust.BodyProperties, ",")), slog.Any("FetchTextBodyValues", cust.FetchTextBodyValues), slog.Any("FetchHTMLBodyValues", cust.FetchHTMLBodyValues), slog.Any("FetchAllBodyValues", cust.FetchAllBodyValues), slog.Any("MaxBodyValueBytes", cust.MaxBodyValueBytes))

	for _, id := range ids {
		idInt64, err := id.Int64()
		if err != nil {
			//the email ids are imap ids meaning they are int64. When they cannot be converted to int64
			//we know that we never going to be able to return them
			notFound = append(notFound, id)
			continue
		}

		em := store.Message{
			ID: idInt64,
		}

		if err := jaccount.DB().Get(ctx, &em); err != nil {
			if err == bstore.ErrAbsent {
				notFound = append(notFound, id)
				continue
			}
			m.mlog.Error("error getting message from db", slog.Any("id", idInt64), slog.Any("error", err.Error()))
			mErr = mlevelerrors.NewMethodLevelErrorServerFail()
			return
		}

		jem, merr := NewEmail(jaccount.Account(), em, m.mlog)
		if merr != nil {
			m.mlog.Error("error instantiating new JEmail", slog.Any("id", idInt64), slog.Any("error", merr.Error()))
			mErr = mlevelerrors.NewMethodLevelErrorServerFail()
			return
		}

		if len(properties) == 0 {
			//no property list found so we return the default set as defined by the standard
			//../../rfc8621:2359
			properties = defaultEmailPropertyFields
		}

		resultElement := Email{
			EmailDefinedProperties: EmailDefinedProperties{
				EmailMetadata: EmailMetadata{
					Id:         jem.Id(),
					ThreadId:   jem.ThreadId(),
					MailboxIds: jem.MailboxIds(),
					Size:       jem.Size(),
					ReceivedAt: jem.ReceivedAt(),
					Keywords:   jem.Keywords(),
					BlobId:     jem.Id(), //FIXME needs review
				},
				EmailBodyParts: EmailBodyParts{
					Preview: "<preview not available>",
				},
			},
			properties: properties,
		}

		var mErrLocal *mlevelerrors.MethodLevelError

		resultElement.MessageId, mErrLocal = jem.MessagedId()
		if mErrLocal != nil {
			m.mlog.Error("error getting messageId", slog.Any("id", idInt64), slog.Any("error", err.Error()))
			mErr = mlevelerrors.NewMethodLevelErrorServerFail()
			return
		}

		resultElement.SentAt, mErrLocal = jem.SendAt()
		if mErrLocal != nil {
			m.mlog.Error("error getting date", slog.Any("id", idInt64), slog.Any("error", err.Error()))

			mErr = mlevelerrors.NewMethodLevelErrorServerFail()
			return
		}

		resultElement.Subject, mErrLocal = jem.Subject()
		if mErrLocal != nil {
			m.mlog.Error("error getting subject", slog.Any("id", idInt64), slog.Any("error", err.Error()))
			mErr = mlevelerrors.NewMethodLevelErrorServerFail()
			return
		}

		resultElement.From, mErrLocal = jem.From()
		if mErrLocal != nil {
			m.mlog.Error("error getting from", slog.Any("id", idInt64), slog.Any("error", err.Error()))
			mErr = mlevelerrors.NewMethodLevelErrorServerFail()
			return
		}

		resultElement.To, mErrLocal = jem.To()
		if mErrLocal != nil {
			m.mlog.Error("error getting to", slog.Any("id", idInt64), slog.Any("error", err.Error()))
			mErr = mlevelerrors.NewMethodLevelErrorServerFail()
			return
		}

		resultElement.CC, mErrLocal = jem.CC()
		if mErrLocal != nil {
			m.mlog.Error("error getting cc", slog.Any("id", idInt64), slog.Any("error", err.Error()))
			mErr = mlevelerrors.NewMethodLevelErrorServerFail()
			return
		}

		resultElement.BCC, mErrLocal = jem.BCC()
		if mErrLocal != nil {
			m.mlog.Error("error getting bcc", slog.Any("id", idInt64), slog.Any("error", err.Error()))
			mErr = mlevelerrors.NewMethodLevelErrorServerFail()
			return
		}

		resultElement.Sender, mErrLocal = jem.Sender()
		if mErrLocal != nil {
			m.mlog.Error("error getting sender", slog.Any("id", idInt64), slog.Any("error", err.Error()))
			mErr = mlevelerrors.NewMethodLevelErrorServerFail()
			return
		}

		resultElement.ReplyTo, mErrLocal = jem.ReplyTo()
		if mErrLocal != nil {
			m.mlog.Error("error getting replyTo", slog.Any("id", idInt64), slog.Any("error", err.Error()))
			mErr = mlevelerrors.NewMethodLevelErrorServerFail()
			return
		}

		resultElement.InReplyTo, mErrLocal = jem.InReplyTo()
		if mErrLocal != nil {
			m.mlog.Error("error getting inReplyTo", slog.Any("id", idInt64), slog.Any("error", mErrLocal.Error()))
			mErr = mlevelerrors.NewMethodLevelErrorServerFail()
			return
		}

		resultElement.Preview, mErrLocal = jem.Preview()
		if mErrLocal != nil {
			m.mlog.Error("error getting preview", slog.Any("id", idInt64), slog.Any("error", mErrLocal.Error()))
			mErr = mlevelerrors.NewMethodLevelErrorServerFail()
			return
		}

		resultElement.References, mErrLocal = jem.References()
		if mErrLocal != nil {
			m.mlog.Error("error getting references", slog.Any("id", idInt64), slog.Any("error", err.Error()))
			mErr = mlevelerrors.NewMethodLevelErrorServerFail()
			return
		}

		for _, prop := range properties {
			if strings.HasPrefix(prop, "header:") {
				//some custom headers are requested
				hParts := strings.Split(prop, ":")

				var headerName string
				var headerFormat string = "raw"
				var returnAll bool

				//if there are only 2 parts, then we use the fallback format which is raw
				if len(hParts) >= 2 {
					headerName = hParts[1]
				}
				if len(hParts) == 3 {
					headerFormat = hParts[2]
				}
				if len(hParts) == 4 {
					if hParts[3] == "all" {
						returnAll = true

					} else {
						continue
					}
				}
				if len(hParts) > 4 {
					//this format we do not recognize to skip it
					continue
				}

				if resultElement.DynamicProperties == nil {
					resultElement.DynamicProperties = make(map[string]any, 1)
				}

				headerInOrder, err := jem.part.HeaderInOrder()
				if err != nil {
					m.mlog.Error("error getting headers", slog.Any("id", idInt64), slog.Any("error", err.Error()))
					mErr = mlevelerrors.NewMethodLevelErrorServerFail()
					return
				}

				resultElement.DynamicProperties[prop], mErrLocal = HeaderAs(headerInOrder, m.mlog, headerName, headerFormat, returnAll)
				if mErrLocal != nil {
					m.mlog.Error("error getting bespoke header", slog.Any("id", idInt64), slog.Any("prop", prop), slog.Any("error", mErr.Error()))
					mErr = mlevelerrors.NewMethodLevelErrorServerFail()
					return
				}
			}
		}

		if HasAny(properties, "bodyStructure") {
			//FIXME In addition, the client may request/send EmailBodyPart properties representing individual header fields, following the same syntax and semantics as for the Email object, e.g., header:Content-Type.
			bs, mErrLocal := jem.BodyStructure(cust.BodyProperties)
			if mErrLocal != nil {
				m.mlog.Error("error getting body structure", slog.Any("id", idInt64), slog.Any("error", mErr.Error()))
				mErr = mlevelerrors.NewMethodLevelErrorServerFail()

			}
			resultElement.BodyStructure = bs
		}

		if HasAny(properties, "bodyValues") {
			bvs, mErrLocal := jem.BodyValues(cust.FetchTextBodyValues, cust.FetchHTMLBodyValues, cust.FetchAllBodyValues, cust.MaxBodyValueBytes)
			if mErrLocal != nil {
				m.mlog.Error("error getting body values", slog.Any("id", idInt64), slog.Any("error", mErr.Error()))
				mErr = mlevelerrors.NewMethodLevelErrorServerFail()
				return
			}
			resultElement.BodyValues = bvs
		}

		if HasAny(properties, "textBody") {
			textBody, mErrLocal := jem.HTMLBody(cust.BodyProperties)
			if mErrLocal != nil {
				m.mlog.Error("error getting textBody", slog.Any("id", idInt64), slog.Any("error", mErr.Error()))
				mErr = mlevelerrors.NewMethodLevelErrorServerFail()
				return
			}
			resultElement.TextBody = textBody
		}

		if HasAny(properties, "htmlBody") {
			htmlBody, mErrLocal := jem.HTMLBody(cust.BodyProperties)
			if mErrLocal != nil {
				m.mlog.Error("error getting htmlBody", slog.Any("id", idInt64), slog.Any("error", mErr.Error()))
				mErr = mlevelerrors.NewMethodLevelErrorServerFail()
				return
			}
			resultElement.HTMLBody = htmlBody
		}

		if HasAny(properties, "attachments") {
			attachments, mErrLocal := jem.Attachments(cust.BodyProperties)
			if mErrLocal != nil {
				m.mlog.Error("error getting attachments", slog.Any("id", idInt64), slog.Any("error", mErr.Error()))
				mErr = mlevelerrors.NewMethodLevelErrorServerFail()
				return
			}
			resultElement.Attachments = attachments
		}

		if HasAny(properties, "hasAttachment") {
			hasAttachment, mErrLocal := jem.HasAttachment()
			if mErrLocal != nil {
				m.mlog.Error("error getting hasAttachment", slog.Any("id", idInt64), slog.Any("error", mErr.Error()))
				mErr = mlevelerrors.NewMethodLevelErrorServerFail()
				return
			}
			resultElement.HasAttachment = hasAttachment
		}

		if HasAny(properties, "headers") {
			hdrs, mErrLocal := jem.Headers()
			if mErrLocal != nil {
				m.mlog.Error("error getting headers", slog.Any("id", idInt64), slog.Any("error", mErrLocal.Error()))
				mErr = mlevelerrors.NewMethodLevelErrorServerFail()
				return
			}
			resultElement.Headers = hdrs
		}

		list = append(list, resultElement)
	}

	if list == nil {
		//always return an empty slice
		list = []any{}
	}

	if notFound == nil {
		//send an empty array instead of a null value to not break the current way of resolving request references
		notFound = []basetypes.Id{}
	}

	//AO: I chose to get the state at the datatype level because the Email().Get is independent from the state and Email().Get already does a lot of things
	var err error
	state, err = m.state(ctx, jaccount.DB())
	if err != nil {
		m.mlog.Error("error getting state", slog.Any("err", err.Error()))
		return accountId, "", nil, notFound, mlevelerrors.NewMethodLevelErrorServerFail()
	}

	return accountId, state, list, notFound, mErr
}

// https://datatracker.ietf.org/doc/html/rfc8620#section-5.3
func (m EmailDT) Set(ctx context.Context, jaccount capabilitier.JAccounter, accountId basetypes.Id, ifInState *string, create map[basetypes.Id]interface{}, update map[basetypes.Id]basetypes.PatchObject, destroy []basetypes.Id) (retAccountId basetypes.Id, oldState *string, newState string, created map[basetypes.Id]interface{}, updated map[basetypes.Id]interface{}, destroyed map[basetypes.Id]interface{}, notCreated map[basetypes.Id]mlevelerrors.SetError, notUpdated map[basetypes.Id]mlevelerrors.SetError, notDestroyed map[basetypes.Id]mlevelerrors.SetError, mErr *mlevelerrors.MethodLevelError) {

	retAccountId = accountId

	m.mlog.Error("SetEmail has not been implemented yet. we just pretend updating went fine")

	updated = make(map[basetypes.Id]interface{}, len(update))

	for updatedId := range update {
		updated[updatedId] = nil
	}
	return
}

// https://datatracker.ietf.org/doc/html/rfc8620#section-5.2
func (m EmailDT) Changes(ctx context.Context, jaccount capabilitier.JAccounter, accountId basetypes.Id, sinceState string, maxChanges *basetypes.Uint) (retAccountId basetypes.Id, oldState string, newState string, hasMoreChanges bool, created, updated, destroyed []basetypes.Id, mErr *mlevelerrors.MethodLevelError) {
	//AO: I am starting to question the goal of splitting the implementation between the datatype/capability layer and the jaccount layer
	//the reason to start with JAccount is not directly hack into the store package and be protected from a lot of changes there
	//however, that is main relevant for the Email/get method
	//also one other goal is to make this testable especially when it comes to mocking bstore. But why not just do an in memory bestore with bogus data? I want to abstract the store in jmap
	//but why not wait till the application is ready for that?
	//return capabilitier.Email().Changes(ctx, accountId, sinceState, maxChanges)

	sinceStateInt64, err := strconv.ParseInt(sinceState, 10, 64)
	if err != nil {
		m.mlog.Error("invalid sinceState: not an int64", slog.Any("err", err.Error()))
		return accountId, sinceState, "", false, nil, nil, nil, mlevelerrors.NewMethodLevelErrorCannotCalculateChanges()
	}

	updatedOrDeletedQ := bstore.QueryDB[store.Message](ctx, jaccount.DB())
	defer updatedOrDeletedQ.Close()
	updatedOrDeletedQ.FilterGreater("ModSeq", store.ModSeqFromClient(sinceStateInt64))
	updatedOrDeletedQ.SortAsc("ModSeq")

	changeBuilder := NewChangeResultBuilder()

	for {
		msg, err := updatedOrDeletedQ.Next()
		if err == bstore.ErrAbsent || (maxChanges != nil && (len(destroyed)+len(updated) > int(*maxChanges.ToPUint()))) {
			//do not get more changes then we need
			break
		}

		if msg.Expunged {
			changeBuilder.AddDestroyed(msg.ID, msg.ModSeq.Client())
		} else {
			changeBuilder.AddUpdated(msg.ID, msg.ModSeq.Client())
		}
	}

	newQ := bstore.QueryDB[store.Message](ctx, jaccount.DB())
	defer newQ.Close()
	newQ.FilterGreater("CreateSeq", store.ModSeqFromClient(sinceStateInt64))
	for {
		msg, err := newQ.Next()
		if err == bstore.ErrAbsent || (maxChanges != nil && (len(created) > int(*maxChanges.ToPUint()))) {
			//do not get more changes then we need
			break
		}
		changeBuilder.AddCreated(msg.ID, msg.ModSeq.Client())

	}

	if len(changeBuilder.Elements) == 0 {
		//no changes so newState = oldState
		return accountId, sinceState, sinceState, false, nil, nil, nil, nil
	}

	hasMoreChanges = maxChanges != nil && len(changeBuilder.Elements) > int(*maxChanges.ToPUint())

	//newState can be the 'final' state or an intermediate state
	created, updated, destroyed, newState = changeBuilder.Final(maxChanges.ToPUint())

	return accountId, sinceState, newState, hasMoreChanges, created, updated, destroyed, nil
}

func (EmailDT) state(ctx context.Context, db *bstore.DB) (string, error) {
	ss := store.SyncState{
		ID: 1,
	}

	if err := db.Get(ctx, &ss); err != nil {
		if err == bstore.ErrAbsent {
			//Email modseqs start at 2 for first assignment so return 1 here is safe
			return "1", nil
		}
	}
	return strconv.FormatInt(ss.LastModSeq.Client(), 10), nil

}

// DownloadBlob returns the raw contents of a blobid. The first param in the reponse indicates if the blob was found
func (m EmailDT) DownloadBlob(ctx context.Context, mAccount *store.Account, blobID, name, Type string) (bool, io.Reader, error) {
	msgID, partID, ok := strings.Cut(blobID, "-")
	if !ok {
		return false, nil, MalformedBlodID
	}

	msgIDint, err := strconv.ParseInt(msgID, 10, 64)
	if err != nil {
		return false, nil, MalformedBlodID
	}

	em := store.Message{
		ID: msgIDint,
	}

	if err := mAccount.DB.Get(ctx, &em); err != nil {
		if err == bstore.ErrAbsent {
			return false, nil, nil
		}
		return false, nil, err
	}

	jem, merr := NewEmail(mAccount, em, m.mlog)
	if merr != nil {
		m.mlog.Error("error instantiating new JEmail", slog.Any("id", msgIDint), slog.Any("error", merr.Error()))
		return false, nil, merr
	}

	jPart, err := jem.GetJPart(partID)
	if err != nil {
		return false, nil, err
	}
	if jPart == nil {
		return false, nil, nil
	}

	return true, jPart.Reader(), nil

}
