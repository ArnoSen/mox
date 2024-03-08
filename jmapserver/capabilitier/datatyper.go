package capabilitier

import (
	"context"
	"encoding/json"

	"github.com/mjl-/mox/jmapserver/basetypes"
	"github.com/mjl-/mox/jmapserver/jaccount"
	"github.com/mjl-/mox/jmapserver/mlevelerrors"
)

type Datatyper interface {
	Name() string
}

type AddedItem struct {
	Id    basetypes.Id
	Index basetypes.Uint
}

type Echoer interface {
	Echo(ctx context.Context, content json.RawMessage) (resp map[string]any, mErr *mlevelerrors.MethodLevelError)
}

type Getter interface {
	//https://datatracker.ietf.org/doc/html/rfc8620#section-5.1
	Get(ctx context.Context, jaccount jaccount.JAccounter, accountId basetypes.Id, ids []basetypes.Id, properties []string, customParams any) (retAccountId basetypes.Id, state string, list []any, notFound []basetypes.Id, mErr *mlevelerrors.MethodLevelError)
	CustomGetRequestParams() any
}

type Changeser interface {
	//https://datatracker.ietf.org/doc/html/rfc8620#section-5.2
	Changes(ctx context.Context, jaccount jaccount.JAccounter, accountId basetypes.Id, sinceState string, maxChanges *basetypes.Uint) (retAccountId basetypes.Id, oldState, newState string, hasMoreChanges bool, created, updated, destroyed []basetypes.Id, mErr *mlevelerrors.MethodLevelError)
}

type Setter interface {
	//https://datatracker.ietf.org/doc/html/rfc8620#section-5.3
	Set(ctx context.Context, jaccount jaccount.JAccounter, accountId basetypes.Id, ifInState *string, create map[basetypes.Id]any, update map[basetypes.Id]basetypes.PatchObject, destroy []basetypes.Id) (retAccountId basetypes.Id, oldState *string, newState string, created, updated, destroyed map[basetypes.Id]any, notCreated, notUpdated, notDestroyed map[basetypes.Id]mlevelerrors.SetError, mErr *mlevelerrors.MethodLevelError)
}

type Copier interface {
	//https://datatracker.ietf.org/doc/html/rfc8620#section-5.4
	Copy(ctx context.Context, fromAccountId basetypes.Id, ifFromState *string, accountId basetypes.Id, ifInState *string, create map[basetypes.Id]interface{}, onSuccessDestroyOriginal bool, destroyFromIfInState *string) (retFromAccountId, retAccountId basetypes.Id, oldState *string, newState string, created map[basetypes.Id]any, notCreated map[basetypes.Id]mlevelerrors.SetError, mErr *mlevelerrors.MethodLevelError)
}

type Querier interface {
	//https://datatracker.ietf.org/doc/html/rfc8620#section-5.5
	Query(ctx context.Context, jaccount jaccount.JAccounter, accountId basetypes.Id, filter *basetypes.Filter, sort []basetypes.Comparator, position basetypes.Int, anchor *basetypes.Id, anchorOffset basetypes.Int, limit *basetypes.Uint, calculateTotal bool, customParams any) (retAccountId basetypes.Id, queryState string, canCalculateChanges bool, retPosition basetypes.Int, ids []basetypes.Id, total basetypes.Uint, retLimit basetypes.Uint, mErr *mlevelerrors.MethodLevelError)
	CustomQueryRequestParams() any
}

type QueryChangeser interface {
	//https://datatracker.ietf.org/doc/html/rfc8620#section-5.6
	QueryChanges(ctx context.Context, accountId basetypes.Id, filter *basetypes.Filter, sort []basetypes.Comparator, sinceQueryState string, maxChanges *basetypes.Uint, upToId *basetypes.Id, calculateTotal bool) (retAccountId basetypes.Id, oldQueryState, newQueryState string, total basetypes.Uint, removed []basetypes.Id, added []AddedItem, mErr *mlevelerrors.MethodLevelError)
}
