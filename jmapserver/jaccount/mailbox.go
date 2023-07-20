package jaccount

import (
	"github.com/mjl-/mox/jmapserver/basetypes"
)

type Mailbox struct {
	Id            basetypes.Id   `json:"id"`
	Name          string         `json:"name"`
	ParentId      *basetypes.Id  `json:"parentId"`
	Role          string         `json:"role"`
	SortOrder     basetypes.Uint `json:"sortOrder"`
	TotalEmails   basetypes.Uint `json:"totalEmails"`
	UnreadEmails  basetypes.Uint `json:"unreadEmails"`
	TotalThreads  basetypes.Uint `json:"totalThreads"`
	UnreadThreads basetypes.Uint `json:"unreadThreads"`
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
