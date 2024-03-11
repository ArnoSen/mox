package mailcapability

import (
	"fmt"
	"strings"

	"github.com/mjl-/mox/store"
)

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
