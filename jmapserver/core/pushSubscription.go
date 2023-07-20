package core

import (
	"context"

	"github.com/mjl-/mox/jmapserver/basetypes"
	"github.com/mjl-/mox/jmapserver/datatyper"
	"github.com/mjl-/mox/jmapserver/mlevelerrors"
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

// https://datatracker.ietf.org/doc/html/rfc8620#section-5.1
func (dps DatatypePushSubscription) Get(ctx context.Context, accountId basetypes.Id, ids []basetypes.Id, properties []string) (retAccountId basetypes.Id, state string, list []interface{}, notFound []basetypes.Id, mErr *mlevelerrors.MethodLevelError) {
	panic("not implemented") // TODO: Implement
}

// https://datatracker.ietf.org/doc/html/rfc8620#section-5.3
func (dps DatatypePushSubscription) Set(ctx context.Context, accountId basetypes.Id, ifInState *string, create map[basetypes.Id]interface{}, update map[basetypes.Id][]datatyper.PatchObject, destroy []basetypes.Id) (retAccountId basetypes.Id, oldState *string, newState string, created map[basetypes.Id]interface{}, updated map[basetypes.Id]interface{}, destroyed map[basetypes.Id]interface{}, notCreated map[basetypes.Id]mlevelerrors.SetError, notUpdated map[basetypes.Id]mlevelerrors.SetError, notDestroyed map[basetypes.Id]mlevelerrors.SetError, mErr *mlevelerrors.MethodLevelError) {
	panic("not implemented") // TODO: Implement
}
