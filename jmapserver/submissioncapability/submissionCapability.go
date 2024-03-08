package submission

import (
	"github.com/mjl-/mox/jmapserver/basetypes"
	"github.com/mjl-/mox/jmapserver/capabilitier"
)

type SubmissionCapabilitySettings struct {
	MaxDelayedSend       basetypes.Uint `json:"maxDelayedSend"`
	SubmissionExtensions [][]string     `json:"submissionExtensions"`
}

type Submission struct {
	settings  SubmissionCapabilitySettings
	datatypes []capabilitier.Datatyper
}

func NewSubmissionCapability(settings SubmissionCapabilitySettings) *Submission {
	return &Submission{
		settings: settings,
		datatypes: []capabilitier.Datatyper{
			NewIdentities(),
			NewEmailSubmission(),
		},
	}
}

func (c Submission) Urn() string {
	return "urn:ietf:params:jmap:submission"
}

func (c *Submission) SessionObjectInfo() interface{} {
	return c.settings
}

func (c *Submission) Datatypes() []capabilitier.Datatyper {
	return c.datatypes
}
