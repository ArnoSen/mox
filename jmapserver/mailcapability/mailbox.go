package mailcapability

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/mjl-/bstore"
	"github.com/mjl-/mox/jmapserver/basetypes"
	"github.com/mjl-/mox/jmapserver/capabilitier"
	"github.com/mjl-/mox/jmapserver/mlevelerrors"
	"github.com/mjl-/mox/mlog"
	"github.com/mjl-/mox/store"
)

type MailboxDT struct {
	//contextUserKey is the key in the context containing the user object
	mlog mlog.Log
}

func NewMailBox(mlog mlog.Log) MailboxDT {
	return MailboxDT{
		mlog: mlog,
	}
}

func (mb MailboxDT) Name() string {
	return "Mailbox"
}

// https://datatracker.ietf.org/doc/html/rfc8620#section-5.1
func (mb MailboxDT) Get(ctx context.Context, jaccount capabilitier.JAccounter, accountId basetypes.Id, ids []basetypes.Id, properties []string, customParams any) (retAccountId basetypes.Id, state string, list []interface{}, notFound []basetypes.Id, mErr *mlevelerrors.MethodLevelError) {

	mbs, err := bstore.QueryDB[store.Mailbox](ctx, jaccount.DB()).List()
	if err != nil {
		mb.mlog.Error("error querying mailboxes", slog.Any("err", err.Error()))
		return accountId, "", nil, nil, mlevelerrors.NewMethodLevelErrorServerFail()
	}

	var result []any

	//put in a structure so we can do sorting
	jmbs := NewJMailboxes(store.MailboxHierarchyDelimiter)

	for _, mb := range mbs {
		jmbs.AddMailbox(NewJMailbox(mb))
	}

loopmailboxes:
	for i, jmb := range jmbs.Mbs {

		if len(ids) > 0 {
			//we only need selected mailboxes
			var mustBeIncluded = false
			for _, id := range ids {
				if string(id) == jmb.ID() {
					mustBeIncluded = true
					break
				}
			}
			if !mustBeIncluded {
				continue loopmailboxes
			}
		}

		resultItem := Mailbox{
			Id:   basetypes.Id(jmb.ID()),
			Name: jmb.Name(),
			//FIXME this should be persistent and come from db
			SortOrder:     basetypes.Uint(uint(i + 1)),
			TotalThreads:  0, //FIXME
			UnreadThreads: 0, //FIXME
			Role:          jmb.Role(),
			MyRights: MailboxRights{
				MayReadItems:   jmb.MayReadItems(),
				MayAddItems:    jmb.MayAddItems(),
				MayRemoveItems: jmb.MayRemoveItems(),
				MaySetSeen:     jmb.MaySetSeen(),
				MaySetKeywords: jmb.MaySetKeywords(),
				MayCreateChild: jmb.MayCreateChild(),
				MayRename:      jmb.MayRename(),
				MayDelete:      jmb.MayDelete(),
				MaySubmit:      jmb.MaySubmit(),
			},
			//Check with MJL
			IsSubscribed: jmb.Subscribed(),
		}
		if jmb.Mb.HaveCounts {
			resultItem.TotalEmails = basetypes.Uint(jmb.TotalEmails())
			resultItem.UnreadEmails = basetypes.Uint(jmb.UnreadEmails())
		}

		if pID := jmbs.ParentID(jmb); pID != nil {
			resultItem.Id = basetypes.Id(*pID)
		}

		result = append(result, resultItem)
	}

	if notFound == nil {
		//notFound cannot be null
		notFound = []basetypes.Id{}
	}

	mbState, err := mb.state(ctx, jaccount.DB())
	if err != nil {
		mb.mlog.Error("error getting mailbox state", slog.Any("err", err.Error()))
		return accountId, "", nil, nil, mlevelerrors.NewMethodLevelErrorServerFail()
	}

	return accountId, mbState, result, notFound, nil
}

// https://datatracker.ietf.org/doc/html/rfc8620#section-5.2
func (mb MailboxDT) Changes(ctx context.Context, jaccount capabilitier.JAccounter, accountId basetypes.Id, sinceState string, maxChanges *basetypes.Uint) (retAccountId basetypes.Id, oldState string, newState string, hasMoreChanges bool, created, updated, destroyed []basetypes.Id, mErr *mlevelerrors.MethodLevelError) {
	//TODO need to add modseq and createseq for mailboxes
	//Without changing the data model, I can only report on updated mailboxes

	sinceStateInt64, err := strconv.ParseInt(sinceState, 10, 64)
	if err != nil {
		mb.mlog.Error("invalid sinceState: not an int64", slog.Any("err", err.Error()))
		return accountId, sinceState, "", false, nil, nil, nil, mlevelerrors.NewMethodLevelErrorCannotCalculateChanges()
	}

	currentState, err := mb.state(ctx, jaccount.DB())
	if err != nil {
		mb.mlog.Error("error getting mailbox state", slog.Any("err", err.Error()))
		return accountId, sinceState, "", false, nil, nil, nil, mlevelerrors.NewMethodLevelErrorServerFail()
	}

	//Get the mailboxes
	mbs, err := bstore.QueryDB[store.Mailbox](ctx, jaccount.DB()).List()
	if err != nil {
		mb.mlog.Error("error querying mailboxes", slog.Any("err", err.Error()))
		return accountId, sinceState, currentState, true, nil, nil, nil, mlevelerrors.NewMethodLevelErrorServerFail()
	}

	for _, mailbox := range mbs {

		var highestModSeqInMB int64

		//get the message with the highest modseq per mailbox
		queryHighestModSeq := bstore.QueryDB[store.Message](ctx, jaccount.DB())
		queryHighestModSeq.FilterGreater("ModSeq", sinceStateInt64)
		queryHighestModSeq.SortDesc("ModSeq")
		queryHighestModSeq.Limit(1)
		queryHighestModSeq.FilterEqual("MailboxID", mailbox.ID)

		msg, err := queryHighestModSeq.Get()
		if err != nil && err != bstore.ErrAbsent {
			mb.mlog.Error("error querying messages", slog.Any("err", err.Error()))
			return accountId, sinceState, currentState, true, nil, nil, nil, mlevelerrors.NewMethodLevelErrorServerFail()
		}
		if err == nil {
			highestModSeqInMB = msg.ModSeq.Client()
		}

		//get highest create seq
		queryHighestCreateSeq := bstore.QueryDB[store.Message](ctx, jaccount.DB())
		queryHighestCreateSeq.FilterGreater("CreateSeq", sinceStateInt64)
		queryHighestCreateSeq.Limit(1)
		queryHighestCreateSeq.SortDesc("CreateSeq")
		queryHighestCreateSeq.FilterEqual("MailboxID", mailbox.ID)

		msg, err = queryHighestCreateSeq.Get()
		if err != nil && err != bstore.ErrAbsent {
			mb.mlog.Error("error querying messages", slog.Any("err", err.Error()))
			return accountId, sinceState, currentState, true, nil, nil, nil, mlevelerrors.NewMethodLevelErrorServerFail()

		}
		if err == nil {
			if msg.CreateSeq.Client() > sinceStateInt64 || highestModSeqInMB > sinceStateInt64 {
				//FIXME need to take into account maxChanges
				updated = append(updated, basetypes.NewIdFromInt64(mailbox.ID))
			}
		}
	}

	return accountId, sinceState, currentState, false, nil, updated, nil, nil
}

func (m MailboxDT) CustomGetRequestParams() any {
	return nil
}

func (_ MailboxDT) state(ctx context.Context, db *bstore.DB) (string, error) {
	//mail box state is the same as email state for now
	return EmailDT{}.state(ctx, db)
}

// ../../rfc/8621:485
type Mailbox struct {
	Id            basetypes.Id   `json:"id"`
	Name          string         `json:"name"`
	ParentId      *basetypes.Id  `json:"parentId"`
	Role          *string        `json:"role"`
	SortOrder     basetypes.Uint `json:"sortOrder"`
	TotalEmails   basetypes.Uint `json:"totalEmails"`
	UnreadEmails  basetypes.Uint `json:"unreadEmails"`
	TotalThreads  basetypes.Uint `json:"totalThreads"`
	UnreadThreads basetypes.Uint `json:"unreadThreads"`
	MyRights      MailboxRights  `json:"myRights"`
	IsSubscribed  bool           `json:"isSubscribed"`
}

// ../../rfc/8621:623
type MailboxRights struct {
	MayReadItems   bool `json:"mayReadItems"`
	MayAddItems    bool `json:"mayAddItems"`
	MayRemoveItems bool `json:"mayRemoveItems"`
	MaySetSeen     bool `json:"maySetSeen"`
	MaySetKeywords bool `json:"maySetKeywords"`
	MayCreateChild bool `json:"mayCreateChild"`
	MayRename      bool `json:"mayRename"`
	MayDelete      bool `json:"mayDelete"`
	MaySubmit      bool `json:"maySubmit"`
}
