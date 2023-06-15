package submission

import (
	"github.com/mjl-/mox/jmapserver/capabilitier"
	"github.com/mjl-/mox/jmapserver/datatyper"
)

type SubmissionCapabilitySettings struct {
	MaxDelayedSend       datatyper.Uint `json:"maxDelayedSend"`
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
