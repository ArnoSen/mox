package mail

import (
	"github.com/mjl-/mox/jmapserver/capabilitier"
	"github.com/mjl-/mox/jmapserver/datatyper"
)

type MailCapabilitySettings struct {
	MaxMailboxesPerEmail       *datatyper.Uint `json:"maxMailboxesPerEmail"`
	MaxMailboxDepth            *datatyper.Uint `json:"maxMailboxDepth"`
	MaxSizeMailboxName         datatyper.Uint  `json:"maxSizeMailboxName"`
	MaxSizeAttachmentsPerEmail datatyper.Uint  `json:"maxSizeAttachmentsPerEmail"`
	EmailQuerySortOptions      []string        `json:"emailQuerySortOptions"`
	MayCreateTopLevelMailbox   bool            `json:"mayCreateTopLevelMailbox"`
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
	return "urn:ietf:params:jmap:mail"
}

func (c *Mail) SessionObjectInfo() interface{} {
	return c.settings
}

func (c *Mail) Datatypes() []capabilitier.Datatyper {
	return c.datatypes
}
