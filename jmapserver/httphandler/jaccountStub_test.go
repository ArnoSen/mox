package httphandler

import (
	"context"

	"github.com/mjl-/mox/jmapserver/basetypes"
	"github.com/mjl-/mox/jmapserver/jaccount"
	"github.com/mjl-/mox/jmapserver/mlevelerrors"
)

type JAccountStub struct {
}

func NewJAccountStub() *JAccountStub {
	return &JAccountStub{}
}

func (jas JAccountStub) Email() jaccount.AccountEmailer {
	return AccountEmailStub{}
}

func (jas JAccountStub) Mailbox() jaccount.AccountMailboxer {
	return AccountMailboxStub{}
}

func (jas JAccountStub) Thread() jaccount.AccountThreader {
	return AccountThreadStub{}
}

type AccountMailboxStub struct {
}

// Mailbox
func (jas AccountMailboxStub) Get(ctx context.Context, ids []basetypes.Id) ([]jaccount.Mailbox, []basetypes.Id, string, *mlevelerrors.MethodLevelError) {
	panic("not implemented") // TODO: Implement
}

type AccountEmailStub struct {
}

// Email
func (jas AccountEmailStub) Get(ctx context.Context, ids []basetypes.Id, properties []string, bodyProperties []string, FetchTextBodyValues bool, FetchHTMLBodyValues bool, FetchAllBodyValues bool, MaxBodyValueBytes *basetypes.Uint) (result []jaccount.Email, notFound []basetypes.Id, mErr *mlevelerrors.MethodLevelError) {
	panic("not implemented") // TODO: Implement
}

func (jas AccountEmailStub) Query(ctx context.Context, filter *basetypes.Filter, sort []basetypes.Comparator, position basetypes.Int, anchor *basetypes.Id, anchorOffset basetypes.Int, limit int, calculateTotal bool, collapseThreads bool) (queryState string, canCalculateChanges bool, retPosition basetypes.Int, ids []basetypes.Id, total basetypes.Uint, mErr *mlevelerrors.MethodLevelError) {
	panic("not implemented") // TODO: Implement
}

func (jas AccountEmailStub) Set(ctx context.Context, ifInState *string, create map[basetypes.Id]interface{}, update map[basetypes.Id]basetypes.PatchObject, destroy []basetypes.Id) (oldState *string, newState string, created map[basetypes.Id]interface{}, updated map[basetypes.Id]interface{}, destroyed map[basetypes.Id]interface{}, notCreated map[basetypes.Id]mlevelerrors.SetError, notUpdated map[basetypes.Id]mlevelerrors.SetError, notDestroyed map[basetypes.Id]mlevelerrors.SetError, mErr *mlevelerrors.MethodLevelError) {
	panic("not implemented") // TODO: Implement
}

func (jas AccountEmailStub) DownloadBlob(ctx context.Context, blobID, name, Type string) (bool, []byte, error) {
	panic("not implemented")
}

func (jas AccountEmailStub) State(ctx context.Context) (string, *mlevelerrors.MethodLevelError) {
	panic("not implemented")
}

type AccountThreadStub struct {
}

// Thread
func (jas AccountThreadStub) Get(ctx context.Context, ids []basetypes.Id) (state string, result []jaccount.Thread, notFound []basetypes.Id, mErr *mlevelerrors.MethodLevelError) {
	panic("not implemented") // TODO: Implement
}

func (jas JAccountStub) Close() error {
	return nil
}
