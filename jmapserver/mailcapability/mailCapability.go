package mailcapability

import (
	"github.com/mjl-/mox/jmapserver/datatyper"
)

const (
	URN = "urn:ietf:params:jmap:mail"
)

type MailCapabilitySettings struct {
	MaxMailboxesPerEmail       *datatyper.Uint `json:"maxMailboxesPerEmail"`
	MaxMailboxDepth            *datatyper.Uint `json:"maxMailboxDepth"`
	MaxSizeMailboxName         datatyper.Uint  `json:"maxSizeMailboxName"`
	MaxSizeAttachmentsPerEmail datatyper.Uint  `json:"maxSizeAttachmentsPerEmail"`
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

func NewMailCapability(settings MailCapabilitySettings) *MailCapability {
	return &MailCapability{
		settings: settings,
		datatypes: []datatyper.Datatyper{
			NewMailBox(),
			NewThread(),
			NewMailBox(),
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
