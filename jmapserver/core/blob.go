package core

import (
	"context"

	"github.com/mjl-/mox/jmapserver/basetypes"
	"github.com/mjl-/mox/jmapserver/mlevelerrors"
)

type DatatypeBlob struct{}

func (dc DatatypeBlob) Name() string {
	return "Blob"
}

func NewDatatypeBlob() DatatypeBlob {
	return DatatypeBlob{}
}

// https://datatracker.ietf.org/doc/html/rfc8620#section-5.4
func (db DatatypeBlob) Copy(ctx context.Context, fromAccountId basetypes.Id, ifFromState *string, accountId basetypes.Id, ifInState *string, create map[basetypes.Id]interface{}, onSuccessDestroyOriginal bool, destroyFromIfInState *string) (retFromAccountId basetypes.Id, retAccountId basetypes.Id, oldState *string, newState string, created map[basetypes.Id]interface{}, notCreated map[basetypes.Id]mlevelerrors.SetError, mErr *mlevelerrors.MethodLevelError) {
	panic("not implemented") // TODO: Implement
}
