package mailcapability

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

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

func (m MailboxDT) Name() string {
	return "Mailbox"
}

// https://datatracker.ietf.org/doc/html/rfc8620#section-5.1
func (mb MailboxDT) Get(ctx context.Context, jaccount capabilitier.JAccounter, accountId basetypes.Id, ids []basetypes.Id, properties []string, customParams any) (retAccountId basetypes.Id, state string, list []interface{}, notFound []basetypes.Id, mErr *mlevelerrors.MethodLevelError) {

	mbs, err := bstore.QueryDB[store.Mailbox](ctx, jaccount.DB()).List()
	if err != nil {
		mb.mlog.Logger.Error("error querying mailboxes", slog.Any("err", err.Error()))
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

	return accountId, "state", result, notFound, nil
}

// https://datatracker.ietf.org/doc/html/rfc8620#section-5.2
func (mb MailboxDT) Changes(ctx context.Context, jaccount capabilitier.JAccounter, accountId basetypes.Id, sinceState string, maxChanges *basetypes.Uint) (retAccountId basetypes.Id, oldState string, newState string, hasMoreChanges bool, created []basetypes.Id, updated []basetypes.Id, destroyed []basetypes.Id, mErr *mlevelerrors.MethodLevelError) {
	//TODO need to add modseq for mailboxes
	//AO: not sure what to send back with regards to oldstate/newstate

	mErr = mlevelerrors.NewMethodLevelErrorCannotCalculateChanges()
	return
}

func (m MailboxDT) CustomGetRequestParams() any {
	return nil
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

type AccountMailbox struct {
	mAccount *store.Account
	mlog     mlog.Log
}

func NewAccountMailbox(mAccount *store.Account, mlog mlog.Log) *AccountMailbox {
	return &AccountMailbox{
		mAccount: mAccount,
		mlog:     mlog,
	}
}

func (ja *AccountMailbox) Get(ctx context.Context, ids []basetypes.Id) (result []Mailbox, notFound []basetypes.Id, state string, mErr *mlevelerrors.MethodLevelError) {

	mbs, err := bstore.QueryDB[store.Mailbox](ctx, ja.mAccount.DB).List()
	if err != nil {
		ja.mlog.Logger.Error("error querying mailboxes", slog.Any("err", err.Error()))
		return nil, nil, "", mlevelerrors.NewMethodLevelErrorServerFail()
	}

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
	return result, notFound, "stubState", nil
}

// JMailbox is a mailbox that contains all the info that JMAP needs for a Mailbox
type JMailbox struct {
	Mb store.Mailbox
}

func NewJMailbox(mb store.Mailbox) JMailbox {
	return JMailbox{
		Mb: mb,
	}
}

func (mb JMailbox) ID() string {
	return fmt.Sprintf("%d", mb.Mb.ID)
}

func (mb JMailbox) Name() string {
	return mb.Mb.Name
}

func (mb JMailbox) MayReadItems() bool {
	return true
}

func (mb JMailbox) MayAddItems() bool {
	return true
}

func (mb JMailbox) MayRemoveItems() bool {
	return true
}

func (mb JMailbox) MaySetSeen() bool {
	return true
}

func (mb JMailbox) MaySetKeywords() bool {
	return false
}

func (mb JMailbox) MayCreateChild() bool {
	return true
}

func (mb JMailbox) MayRename() bool {
	return true
}

func (mb JMailbox) MayDelete() bool {
	//do not allow deletion of special mailboxes
	return mb.Role() == nil
}

func (mb JMailbox) MaySubmit() bool {
	return true
}

func (mb JMailbox) Subscribed() bool {
	return true
}

func (mb JMailbox) Role() *string {
	var result string

	switch {
	//see https://www.iana.org/assignments/imap-mailbox-name-attributes/imap-mailbox-name-attributes.xhtml
	// ../../rfc/8621:518

	//FIXME need to confirm from documentation that inbox is always called inbox
	case strings.ToLower(mb.Mb.Name) == "inbox":
		//Inbox is a JMAP only role
		// ../../rfc/8621:518
		result = "Inbox"
	case mb.Mb.SpecialUse.Archive:
		result = "Archive"
	case mb.Mb.SpecialUse.Draft:
		result = "Draft"
	case mb.Mb.SpecialUse.Junk:
		result = "Junk"
	case mb.Mb.SpecialUse.Sent:
		result = "Sent"
	case mb.Mb.SpecialUse.Trash:
		result = "Trash"
	default:
		return nil
	}
	return &result
}

func (mb JMailbox) TotalEmails() uint {
	return uint(mb.Mb.MailboxCounts.Total)
}

func (mb JMailbox) UnreadEmails() uint {
	return uint(mb.Mb.MailboxCounts.Unread)
}

type JMailboxes struct {
	Mbs                []JMailbox
	HierarchyDelimiter string
}

func NewJMailboxes(hierarchyDelimiter string, mbs ...JMailbox) JMailboxes {
	return JMailboxes{
		Mbs: mbs,
		//AO: I cannot find a constant in the code describing the hierarchy. I got this from a comment
		HierarchyDelimiter: hierarchyDelimiter,
	}
}

func (jmbs *JMailboxes) AddMailbox(mb JMailbox) {
	jmbs.Mbs = append(jmbs.Mbs, mb)
}

// Parent returns the ID of the parent mailbox
func (jmbs JMailboxes) ParentID(mb JMailbox) *string {
	//mailboxes have names like Inbox|Keep|2022

	parts := strings.Split(mb.Mb.Name, jmbs.HierarchyDelimiter)
	if len(parts) == 1 {
		//no seperator so we are at the top level
		return nil
	}

	//remove the last element to get the parent name
	parentName := strings.Join(parts[:len(parts)-1], jmbs.HierarchyDelimiter)

	for _, mb := range jmbs.Mbs {
		if mb.Mb.Name == parentName {
			pID := fmt.Sprintf("%d", mb.Mb.ID)
			return &pID
		}
	}
	return nil
}
