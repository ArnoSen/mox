package httphandler

import (
	"context"
	"encoding/json"

	"github.com/mjl-/mox/jmapserver/basetypes"
	"github.com/mjl-/mox/jmapserver/datatyper"
	"github.com/mjl-/mox/jmapserver/jaccount"
	"github.com/mjl-/mox/jmapserver/mlevelerrors"
)

type StubCapability struct {
	urn         string
	sessioninfo interface{}
	datatypes   []datatyper.Datatyper
}

func NewStubCapacility(urn string, sessionInfo interface{}, datatypes ...datatyper.Datatyper) *StubCapability {
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

func (sc *StubCapability) Datatypes() []datatyper.Datatyper {
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

var (
	_ datatyper.Getter = StubDatatype{}
	_ datatyper.Setter = StubDatatype{}
)

func (sdt StubDatatype) Name() string {
	return sdt.name
}

func (sdt StubDatatype) CustomGetRequestParams() any {
	return nil
}

func (sdt StubDatatype) Echo(ctx context.Context, content json.RawMessage) (map[string]interface{}, *mlevelerrors.MethodLevelError) {
	var resp map[string]interface{}

	err := json.Unmarshal(content, &resp)
	if err != nil {
		return nil, mlevelerrors.NewMethodLevelErrorInvalidArguments("arguments for echo it not map[string]object")
	}

	return resp, nil
}

func (sdt StubDatatype) Get(ctx context.Context, jac jaccount.JAccounter, accountId basetypes.Id, ids []basetypes.Id, properties []string, customProps any) (retAccountId basetypes.Id, state string, list []interface{}, notFound []basetypes.Id, mErr *mlevelerrors.MethodLevelError) {
	//just return empty values
	retAccountId = accountId
	return
}

// https://datatracker.ietf.org/doc/html/rfc8620#section-5.3
func (sdt StubDatatype) Set(ctx context.Context, jaccount jaccount.JAccounter, accountId basetypes.Id, ifInState *string, create map[basetypes.Id]interface{}, update map[basetypes.Id]basetypes.PatchObject, destroy []basetypes.Id) (retAccountId basetypes.Id, oldState *string, newState string, created map[basetypes.Id]interface{}, updated map[basetypes.Id]interface{}, destroyed map[basetypes.Id]interface{}, notCreated map[basetypes.Id]mlevelerrors.SetError, notUpdated map[basetypes.Id]mlevelerrors.SetError, notDestroyed map[basetypes.Id]mlevelerrors.SetError, mErr *mlevelerrors.MethodLevelError) {
	return
}
