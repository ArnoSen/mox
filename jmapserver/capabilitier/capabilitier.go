package capabilitier

import (
	"github.com/mjl-/mox/jmapserver/core"
	"github.com/mjl-/mox/jmapserver/datatyper"
)

type Capabilitiers []Capabilitier

// GetDatatypeByName gets a datatype by name
func (cs Capabilitiers) GetDatatypeByName(name string) datatyper.Datatyper {
	for _, c := range cs {
		for _, dt := range c.Datatypes() {
			if dt.Name() == name {
				return dt
			}
		}
	}
	return nil
}

func (cs Capabilitiers) CoreSettings() *core.CoreCapabilitySettings {
	for _, c := range cs {
		if c.Urn() == core.URN {
			if coreCapability, ok := c.(*core.Core); ok {
				return &coreCapability.Settings
			}
			panic("no core settings found")
		}
	}
	return nil
}

// Capabilitier needs to be implemented by a JMAP capabality
type Capabilitier interface {
	//Urn for the capability
	Urn() string

	//SessionObjectInfo is data that is added to the session object
	SessionObjectInfo() interface{}

	//Datatypes returns the datatypes associated with the capability
	Datatypes() []datatyper.Datatyper
}
