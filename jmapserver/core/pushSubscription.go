package core

import (
	"context"

	"github.com/mjl-/mox/jmapserver/datatyper"
)

type DatatypePushSubscription struct {
	//implements get & set
}

func NewDatatypePushSubscription() DatatypePushSubscription {
	return DatatypePushSubscription{}
}

func (dps DatatypePushSubscription) Name() string {
	return "PushSubscription"
}

//https://datatracker.ietf.org/doc/html/rfc8620#section-5.1
func (dps DatatypePushSubscription) Get(ctx context.Context, accountId datatyper.Id, ids []datatyper.Id, properties []string) (retAccountId datatyper.Id, state string, list []interface{}, notFound []datatyper.Id, mErr *datatyper.MethodLevelError) {
	panic("not implemented") // TODO: Implement
}

//https://datatracker.ietf.org/doc/html/rfc8620#section-5.3
func (dps DatatypePushSubscription) Set(ctx context.Context, accountId datatyper.Id, ifInState *string, create map[datatyper.Id]interface{}, update map[datatyper.Id][]datatyper.PatchObject, destroy []datatyper.Id) (retAccountId datatyper.Id, oldState *string, newState string, created map[datatyper.Id]interface{}, updated map[datatyper.Id]interface{}, destroyed map[datatyper.Id]interface{}, notCreated map[datatyper.Id]datatyper.SetError, notUpdated map[datatyper.Id]datatyper.SetError, notDestroyed map[datatyper.Id]datatyper.SetError, mErr *datatyper.MethodLevelError) {
	panic("not implemented") // TODO: Implement
}
