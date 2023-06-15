package capabilitier

type Datatyper interface {
	Name() string
}

type Capabilitiers []Capabilitier

//GetDatatypeByName gets a datatype by name
func (cs Capabilitiers) GetDatatypeByName(name string) Datatyper {
	for _, c := range cs {
		for _, dt := range c.Datatypes() {
			if dt.Name() == name {
				return dt
			}
		}
	}
	return nil
}

//Capabilitier needs to be implemented by a JMAP capabality
type Capabilitier interface {
	//Urn for the capability
	Urn() string

	//SessionObjectInfo is data that is added to the session object
	SessionObjectInfo() interface{}

	//Datatypes returns the datatypes associated with the capability
	Datatypes() []Datatyper
}
