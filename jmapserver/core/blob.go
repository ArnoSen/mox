package core

import (
	"context"

	"github.com/mjl-/mox/jmapserver/datatyper"
)

type DatatypeBlob struct{}

func (dc DatatypeBlob) Name() string {
	return "Blob"
}

func NewDatatypeBlob() DatatypeBlob {
	return DatatypeBlob{}
}

//https://datatracker.ietf.org/doc/html/rfc8620#section-5.4
func (db DatatypeBlob) Copy(ctx context.Context, fromAccountId datatyper.Id, ifFromState *string, accountId datatyper.Id, ifInState *string, create map[datatyper.Id]interface{}, onSuccessDestroyOriginal bool, destroyFromIfInState *string) (retFromAccountId datatyper.Id, retAccountId datatyper.Id, oldState *string, newState string, created map[datatyper.Id]interface{}, notCreated map[datatyper.Id]datatyper.SetError, mErr *datatyper.MethodLevelError) {
	panic("not implemented") // TODO: Implement
}
