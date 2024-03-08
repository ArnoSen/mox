package mailcapability

import (
	"github.com/mjl-/mox/jmapserver/basetypes"
	"github.com/mjl-/mox/jmapserver/capabilitier"
	"github.com/mjl-/mox/mlog"
)

// verify Mailbox fulfills getter
var _ capabilitier.Getter = NewMailBox()

const (
	URN = "urn:ietf:params:jmap:mail"

	//NB: this is not an officially documented limit in the RFC
	maxEmailQueryLimit = 50
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
	datatypes []capabilitier.Datatyper
}

func NewMailCapability(settings MailCapabilitySettings, contextUserKey string, logger mlog.Log) *MailCapability {
	return &MailCapability{
		settings: settings,
		datatypes: []capabilitier.Datatyper{
			NewMailBox(),
			NewThread(),
			NewEmailDT(maxEmailQueryLimit, logger),
		},
	}
}

func (c MailCapability) Urn() string {
	return URN
}

func (c *MailCapability) SessionObjectInfo() interface{} {
	return c.settings
}

func (c *MailCapability) Datatypes() []capabilitier.Datatyper {
	return c.datatypes
}
