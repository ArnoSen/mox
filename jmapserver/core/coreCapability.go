package core

import (
	"github.com/mjl-/mox/jmapserver/capabilitier"
)

type Core struct {
	settings  CoreCapabilitySettings
	datatypes []capabilitier.Datatyper
}

func NewCore(settings CoreCapabilitySettings) *Core {
	return &Core{
		settings: settings,
		datatypes: []capabilitier.Datatyper{
			NewDatatypeCore(),
			NewDatatypePushSubscription(),
			NewDatatypeBlob(),
		},
	}
}

func (c Core) Urn() string {
	return "urn:ietf:params:jmap:core"
}

func (c *Core) SessionObjectInfo() interface{} {
	return c.settings
}

func (c *Core) Datatypes() []capabilitier.Datatyper {
	return c.datatypes
}

//CoreCapabilitySettings are the settings for core
//This is passed as response to SessionObjectInfo which is sent without any checks by the session handler so we need the json tags here
type CoreCapabilitySettings struct {
	MaxSizeUpload         uint     `json:"maxSizeUpload"`
	MaxConcurrentUpload   uint     `json:"maxConcurrentUpload"`
	MaxSizeRequest        uint     `json:"maxSizeRequest"`
	MaxConcurrentRequests uint     `json:"maxConcurrentRequests"`
	MaxCallsInRequest     uint     `json:"maxCallsInRequest"`
	MaxObjectsInGet       uint     `json:"maxObjectsInGet"`
	MaxObjectsInSet       uint     `json:"maxObjectsInSet"`
	CollationAlgorithms   []string `json:"collationAlgorithms"`
}
