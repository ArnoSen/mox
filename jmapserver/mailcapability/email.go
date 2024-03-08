package mailcapability

import (
	"context"
	"log/slog"

	"github.com/mjl-/mox/jmapserver/basetypes"
	"github.com/mjl-/mox/jmapserver/jaccount"
	"github.com/mjl-/mox/jmapserver/mlevelerrors"
	"github.com/mjl-/mox/mlog"
)

type EmailDT struct {
	//maxQueryLimit is the number of emails returned for a query request
	maxQueryLimit int
	logger        mlog.Log
}

func NewEmailDT(maxQueryLimit int, logger mlog.Log) EmailDT {
	return EmailDT{
		maxQueryLimit: maxQueryLimit,
		logger:        logger,
	}
}

func (m EmailDT) Name() string {
	return "Email"
}

// https://datatracker.ietf.org/doc/html/rfc8620#section-5.5
func (m EmailDT) Query(ctx context.Context, jaccount jaccount.JAccounter, accountId basetypes.Id, filter *basetypes.Filter, sort []basetypes.Comparator, position basetypes.Int, anchor *basetypes.Id, anchorOffset basetypes.Int, limit *basetypes.Uint, calculateTotal bool, customParams any) (retAccountId basetypes.Id, queryState string, canCalculateChanges bool, retPosition basetypes.Int, ids []basetypes.Id, total basetypes.Uint, retLimit basetypes.Uint, mErr *mlevelerrors.MethodLevelError) {

	//FIXME
	//Need to handle collapseThreads ../../rfc/8621:2506

	var adjustedLimit int = m.maxQueryLimit

	if limit != nil && int(*limit) < adjustedLimit {
		adjustedLimit = int(*limit)
	}

	cust := customParams.(*CustomQueryRequestParams)

	state, canCalculateChanges, retPosition, ids, total, mErr := jaccount.Email().Query(ctx, filter, sort, position, anchor, anchorOffset, adjustedLimit, calculateTotal, cust.CollapseThreads)

	if ids == nil {
		//send an empty array instead of a null value to not break the current way of resolving request references
		ids = []basetypes.Id{}
	}

	return accountId, state, canCalculateChanges, basetypes.Int(retPosition), ids, total, basetypes.Uint(adjustedLimit), mErr
}

type CustomQueryRequestParams struct {
	CollapseThreads bool `json:"collapseThreads"`
}

func (m EmailDT) CustomQueryRequestParams() any {
	return &CustomQueryRequestParams{}
}

type CustomGetRequestParams struct {
	BodyProperties      []string        `json:"bodyProperties"`
	FetchTextBodyValues bool            `json:"fetchTextBodyValues"`
	FetchHTMLBodyValues bool            `json:"fetchHTMLBodyValues"`
	FetchAllBodyValues  bool            `json:"fetchAllBodyValues"`
	MaxBodyValueBytes   *basetypes.Uint `json:"maxBodyValueBytes"`
}

func (m EmailDT) CustomGetRequestParams() any {
	return &CustomGetRequestParams{}
}

// https://datatracker.ietf.org/doc/html/rfc8620#section-5.1
func (m EmailDT) Get(ctx context.Context, jaccount jaccount.JAccounter, accountId basetypes.Id, ids []basetypes.Id, properties []string, customParams any) (retAccountId basetypes.Id, state string, list []any, notFound []basetypes.Id, mErr *mlevelerrors.MethodLevelError) {

	cust := customParams.(*CustomGetRequestParams)

	//property filtering is done at the handler level. It is included here so we can check if some fields are needed in the result
	result, notFound, mErr := jaccount.Email().Get(ctx, ids, properties, cust.BodyProperties, cust.FetchTextBodyValues, cust.FetchHTMLBodyValues, cust.FetchAllBodyValues, cust.MaxBodyValueBytes)

	for _, r := range result {
		list = append(list, r)
	}

	if list == nil {
		//always return an empty slice
		list = []any{}
	}

	if notFound == nil {
		//send an empty array instead of a null value to not break the current way of resolving request references
		notFound = []basetypes.Id{}
	}

	//AO: I chose to get the state at the datatype level because the Email().Get is independent from the state and Email().Get already does a lot of things
	var err error
	state, err = jaccount.Email().State(ctx)
	if err != nil {
		m.logger.Error("error getting state", slog.Any("err", err.Error()))
		return accountId, "", nil, notFound, mlevelerrors.NewMethodLevelErrorServerFail()
	}

	return accountId, state, list, notFound, mErr
}

// https://datatracker.ietf.org/doc/html/rfc8620#section-5.3
func (m EmailDT) Set(ctx context.Context, jaccount jaccount.JAccounter, accountId basetypes.Id, ifInState *string, create map[basetypes.Id]interface{}, update map[basetypes.Id]basetypes.PatchObject, destroy []basetypes.Id) (retAccountId basetypes.Id, oldState *string, newState string, created map[basetypes.Id]interface{}, updated map[basetypes.Id]interface{}, destroyed map[basetypes.Id]interface{}, notCreated map[basetypes.Id]mlevelerrors.SetError, notUpdated map[basetypes.Id]mlevelerrors.SetError, notDestroyed map[basetypes.Id]mlevelerrors.SetError, mErr *mlevelerrors.MethodLevelError) {

	_, newState, _, updated, _, _, _, _, _ = jaccount.Email().Set(ctx, ifInState, create, update, destroy)

	retAccountId = accountId
	return
}

// https://datatracker.ietf.org/doc/html/rfc8620#section-5.2
func (m EmailDT) Changes(ctx context.Context, jaccount jaccount.JAccounter, accountId basetypes.Id, sinceState string, maxChanges *basetypes.Uint) (retAccountId basetypes.Id, oldState string, newState string, hasMoreChanges bool, created, updated, destroyed []basetypes.Id, mErr *mlevelerrors.MethodLevelError) {
	//AO: I am starting to question the goal of splitting the implementation between the datatype/capability layer and the jaccount layer
	//the reason to start with JAccount is not directly hack into the store package and be protected from a lot of changes there
	//however, that is main relevant for the Email/get method
	//also one other goal is to make this testable especially when it comes to mocking bstore. But why not just do an in memory bestore with bogus data? I want to abstract the store in jmap
	//but why not wait till the application is ready for that?
	return jaccount.Email().Changes(ctx, accountId, sinceState, maxChanges)
}
