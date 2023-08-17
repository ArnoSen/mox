package jaccount

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/mjl-/bstore"
	"github.com/mjl-/mox/jmapserver/basetypes"
	"github.com/mjl-/mox/jmapserver/mlevelerrors"
	"github.com/mjl-/mox/store"
)

// JAccount is an adaptor for a mox account. It serves the JMAP specific datatypes
// Ideally this package should be removed over time and all logic should be moved to the mox core packages
// that have knowlegde about what properties are stored in persistent storage and what properties are calculated
type JAccounter interface {
	GetMailboxes(ctx context.Context, ids []basetypes.Id) ([]Mailbox, []basetypes.Id, string, *mlevelerrors.MethodLevelError)
}

type JAccount struct {
	mAccount *store.Account
}

func NewJAccount(mAccount *store.Account) *JAccount {
	return &JAccount{
		mAccount: mAccount,
	}
}

func (ja *JAccount) GetMailboxes(ctx context.Context, ids []basetypes.Id) (result []Mailbox, notFound []basetypes.Id, state string, mErr *mlevelerrors.MethodLevelError) {

	q := bstore.QueryDB[store.Mailbox](ctx, ja.mAccount.DB)

	mbs, err := q.List()
	if err != nil {
		mErr = mlevelerrors.NewMethodLevelErrorServerFail()
		return
	}

	//put in a structure so we can do sorting
	jmbs := NewJMailboxes()

	for _, mb := range mbs {
		jmbs.AddMailbox(NewJMailbox(mb))
	}

	sort.Sort(jmbs)

	for i, jmb := range jmbs.Mbs {

		if len(ids) > 0 {
			//we only need selected mailboxes
			for _, id := range ids {
				if string(id) != jmb.ID() {
					continue
				}
			}
		}

		resultItem := Mailbox{
			Id:            basetypes.Id(jmb.ID()),
			Name:          jmb.Name(),
			SortOrder:     basetypes.Uint(uint(i)),
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
	return
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
	return mb.Role() == ""
}

func (mb JMailbox) MaySubmit() bool {
	return true
}

func (mb JMailbox) Subscribed() bool {
	return true
}

func (mb JMailbox) Role() string {
	//FIXME: inbox is not a special use?
	switch {
	//see https://www.iana.org/assignments/imap-mailbox-name-attributes/imap-mailbox-name-attributes.xhtml
	// ../../rfc/8621:518
	case mb.Mb.SpecialUse.Archive:
		return "Archive"
	case mb.Mb.SpecialUse.Draft:
		return "Draft"
	case mb.Mb.SpecialUse.Junk:
		return "Junk"
	case mb.Mb.SpecialUse.Sent:
		return "Sent"
	case mb.Mb.SpecialUse.Trash:
		return "Trash"
	default:
		return ""
	}
}

func (mb JMailbox) TotalEmails() uint {
	return uint(mb.Mb.MailboxCounts.Total)
}

func (mb JMailbox) UnreadEmails() uint {
	return uint(mb.Mb.MailboxCounts.Unread)
}

type JMailboxes struct {
	Mbs       []JMailbox
	Seperator string
}

func NewJMailboxes(mbs ...JMailbox) JMailboxes {
	return JMailboxes{
		Mbs:       mbs,
		Seperator: "|",
	}
}

func (jmbs JMailboxes) AddMailbox(mb JMailbox) {
	jmbs.Mbs = append(jmbs.Mbs, mb)
}

// Parent returns the ID of the parent mailbox
func (jmbs JMailboxes) ParentID(mb JMailbox) *string {
	//mailboxes have names like Inbox|Keep|2022

	parts := strings.Split(mb.Mb.Name, jmbs.Seperator)
	if len(parts) == 1 {
		//no seperator so we are at the top level
		return nil
	}

	//remove the last element to get the parent name
	parentName := strings.Join(parts[:len(parts)-1], jmbs.Seperator)

	for _, mb := range jmbs.Mbs {
		if mb.Mb.Name == parentName {
			pID := fmt.Sprintf("%d", mb.Mb.ID)
			return &pID
		}
	}
	return nil
}

func (jmbs JMailboxes) IsTopLevel(mb1 JMailbox) bool {
	return !strings.Contains(mb1.Name(), jmbs.Seperator)
}

func (jmbs JMailboxes) ShareAncestor(mb1, mb2 JMailbox) bool {
	if mb1.Name() == "" || mb2.Name() == "" {
		panic("mailbox with empty name!")
	}

	mb1NameParts := strings.Split(mb1.Name(), jmbs.Seperator)
	mb2NameParts := strings.Split(mb2.Name(), jmbs.Seperator)

	if len(mb1NameParts) == 1 && len(mb2NameParts) == 1 {
		//two top level elements
		return false
	}

	return mb1NameParts[0] == mb2NameParts[0]
}

// GetMailboxByID returns a mailbox by ID. If not found, nil is returned
func (jmbs JMailboxes) GetMailboxByID(id string) *JMailbox {
	for _, jmb := range jmbs.Mbs {
		if jmb.ID() == id {
			return &jmb
		}
	}
	return nil
}

func (jmbs JMailboxes) HasSpecialParent(mb1 JMailbox) bool {

	parentID := jmbs.ParentID(mb1)
	if parentID == nil {
		return false
	}

	parent := jmbs.GetMailboxByID(*parentID)
	if parent == nil {
		panic(fmt.Sprintf("mailbox with id %s is missing", *parentID))
	}
	if parent.Role() != "" {
		return true
	}
	return jmbs.HasSpecialParent(*parent)
}

func (jmbs JMailboxes) IsParent(son, father JMailbox) bool {
	parentID := jmbs.ParentID(son)
	if parentID == nil {
		return false
	}
	return *parentID == father.ID()
}

// Len is the number of elements in the collection.
func (jmb JMailboxes) Len() int {
	return len(jmb.Mbs)
}

// Less reports whether the element with index i
// must sort before the element with index j.
//
// If both Less(i, j) and Less(j, i) are false,
// then the elements at index i and j are considered equal.
// Sort may place equal elements in any order in the final result,
// while Stable preserves the original input order of equal elements.
//
// Less must describe a transitive ordering:
//   - if both Less(i, j) and Less(j, k) are true, then Less(i, k) must be true as well.
//   - if both Less(i, j) and Less(j, k) are false, then Less(i, k) must be false as well.
//
// Note that floating-point comparison (the < operator on float32 or float64 values)
// is not a transitive ordering when not-a-number (NaN) values are involved.
// See Float64Slice.Less for a correct implementation for floating-point values.
func (jmbs JMailboxes) Less(i int, j int) (result bool) {
	//FIXME this assumes that special mailboxes are created as first entries in the DB and that the user cannot remove those. Needs a fix for existing users
	return jmbs.Mbs[i].Mb.Name < jmbs.Mbs[j].Mb.Name
}

// Swap swaps the elements with indexes i and j.
func (jmb JMailboxes) Swap(i int, j int) {
	jmb.Mbs[i], jmb.Mbs[j] = jmb.Mbs[j], jmb.Mbs[i]
}
