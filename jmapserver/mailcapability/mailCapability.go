package mailcapability

import (
	"github.com/mjl-/mox/jmapserver/capabilitier"
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

type Mail struct {
	settings  MailCapabilitySettings
	datatypes []capabilitier.Datatyper
}

func NewMail(settings MailCapabilitySettings) *Mail {
	return &Mail{
		settings: settings,
		datatypes: []capabilitier.Datatyper{
			NewMailBox(),
			NewThread(),
			NewMailBox(),
		},
	}
}

func (c Mail) Urn() string {
	return URN
}

func (c *Mail) SessionObjectInfo() interface{} {
	return c.settings
}

func (c *Mail) Datatypes() []capabilitier.Datatyper {
	return c.datatypes
}
