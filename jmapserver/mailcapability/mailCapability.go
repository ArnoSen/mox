package mailcapability

import (
	"github.com/mjl-/mox/jmapserver/basetypes"
	"github.com/mjl-/mox/jmapserver/datatyper"
)

// verify Mailbox fulfills getter
var _ datatyper.Getter = NewMailBox()

const (
	URN = "urn:ietf:params:jmap:mail"
)

type MailCapabilitySettings struct {
	MaxMailboxesPerEmail       *basetypes.Uint `json:"maxMailboxesPerEmail"`
	MaxMailboxDepth            *basetypes.Uint `json:"maxMailboxDepth"`
	MaxSizeMailboxName         basetypes.Uint  `json:"maxSizeMailboxName"`
	MaxSizeAttachmentsPerEmail basetypes.Uint  `json:"maxSizeAttachmentsPerEmail"`
	EmailQuerySortOptions      []string        `json:"emailQuerySortOptions"`
	MayCreateTopLevelMailbox   bool            `json:"mayCreateTopLevelMailbox"`
}

// NewDefaultMailCapabilitySettings is a stub that is used in the session endpoint
func NewDefaultMailCapabilitySettings() MailCapabilitySettings {
	return MailCapabilitySettings{
		MaxSizeMailboxName:         10,
		MaxSizeAttachmentsPerEmail: 100000,
		EmailQuerySortOptions:      []string{},
		MayCreateTopLevelMailbox:   false,
	}
}

type MailCapability struct {
	settings  MailCapabilitySettings
	datatypes []datatyper.Datatyper
}

func NewMailCapability(settings MailCapabilitySettings, contextUserKey string) *MailCapability {
	return &MailCapability{
		settings: settings,
		datatypes: []datatyper.Datatyper{
			NewMailBox(),
			NewThread(),
			NewEmail(),
		},
	}
}

func (c MailCapability) Urn() string {
	return URN
}

func (c *MailCapability) SessionObjectInfo() interface{} {
	return c.settings
}

func (c *MailCapability) Datatypes() []datatyper.Datatyper {
	return c.datatypes
}
