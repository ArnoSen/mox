package core

import (
	"context"
	"encoding/json"

	"github.com/mjl-/mox/jmapserver/mlevelerrors"
)

type DatatypeCore struct {
	//implements echo
}

func NewDatatypeCore() DatatypeCore {
	return DatatypeCore{}
}

func (dc DatatypeCore) Name() string {
	return "Core"
}

func (dc DatatypeCore) Echo(ctx context.Context, content json.RawMessage) (resp map[string]interface{}, mErr *mlevelerrors.MethodLevelError) {
	panic("not implemented") // TODO: Implement
}
