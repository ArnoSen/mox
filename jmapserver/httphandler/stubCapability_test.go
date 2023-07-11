package httphandler

import (
	"context"
	"encoding/json"

	"github.com/mjl-/mox/jmapserver/capabilitier"
	"github.com/mjl-/mox/jmapserver/datatyper"
)

type StubCapability struct {
	urn         string
	sessioninfo interface{}
	datatypes   []capabilitier.Datatyper
}

func NewStubCapacility(urn string, sessionInfo interface{}, datatypes ...capabilitier.Datatyper) *StubCapability {
	return &StubCapability{
		urn:         urn,
		sessioninfo: sessionInfo,
		datatypes:   datatypes,
	}
}

func (sc *StubCapability) Urn() string {
	return sc.urn
}

func (sc *StubCapability) SessionObjectInfo() interface{} {
	return sc.sessioninfo
}

func (sc *StubCapability) Datatypes() []capabilitier.Datatyper {
	return sc.datatypes
}

type StubDatatype struct {
	name string
}

func NewStubDatatype(name string) StubDatatype {
	return StubDatatype{
		name: name,
	}
}

func (sdt StubDatatype) Name() string {
	return sdt.name
}

func (sdt StubDatatype) Echo(ctx context.Context, content json.RawMessage) (map[string]interface{}, *datatyper.MethodLevelError) {
	var resp map[string]interface{}

	err := json.Unmarshal(content, &resp)
	if err != nil {
		return nil, datatyper.NewMethodLevelErrorInvalidArguments("arguments for echo it not map[string]object")
	}

	return resp, nil
}

func (sdt StubDatatype) Get(ctx context.Context, accountId datatyper.Id, ids []datatyper.Id, properties []string) (retAccountId datatyper.Id, state string, list []interface{}, notFound []datatyper.Id, mErr *datatyper.MethodLevelError) {
	//just return empty values
	retAccountId = accountId
	return
}

func (sdt StubDatatype) Set(ctx context.Context, accountId datatyper.Id, ifInState *string, create map[datatyper.Id]interface{}, update map[datatyper.Id][]datatyper.PatchObject, destroy []datatyper.Id) (retAccountId datatyper.Id, oldState *string, newState string, created, updated, destroyed map[datatyper.Id]interface{}, notCreated, notUpdated, notDestroyed map[datatyper.Id]datatyper.SetError, mErr *datatyper.MethodLevelError) {
	//just return empty values
	return
}
