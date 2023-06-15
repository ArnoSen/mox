package submission

type Identities struct {
}

func NewIdentities() Identities {
	return Identities{}
}

func (m Identities) Name() string {
	return "Identities"
}
