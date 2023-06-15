package mail

import "github.com/mjl-/mox/jmapserver/datatyper"

type MailboxDT struct {
}

func NewMailBox() MailboxDT {
	return MailboxDT{}
}

func (m MailboxDT) Name() string {
	return "Mailbox"
}

type Mailbox struct {
	Id            datatyper.Id   `json:"id"`
	Name          string         `json:"name"`
	ParentId      *datatyper.Id  `json:"parentId"`
	Role          string         `json:"role"`
	SortOrder     datatyper.Uint `json:"sortOrder"`
	TotalEmails   datatyper.Uint `json:"totalEmails"`
	UnreadEmails  datatyper.Uint `json:"unreadEmails"`
	TotalThreads  datatyper.Uint `json:"totalThreads"`
	UnreadThreads datatyper.Uint `json:"unreadThreads"`
	MyRights      MailboxRights  `json:"myRights"`
	IsSubscribed  bool           `json:"isSubscribed"`
}

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
