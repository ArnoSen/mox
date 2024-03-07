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

// Mailbox
func (jas JAccountStub) GetMailboxes(ctx context.Context, ids []basetypes.Id) ([]jaccount.Mailbox, []basetypes.Id, string, *mlevelerrors.MethodLevelError) {
	panic("not implemented") // TODO: Implement
}

// Email
func (jas JAccountStub) GetEmail(ctx context.Context, ids []basetypes.Id, properties []string, bodyProperties []string, FetchTextBodyValues bool, FetchHTMLBodyValues bool, FetchAllBodyValues bool, MaxBodyValueBytes *basetypes.Uint) (state string, result []jaccount.Email, notFound []basetypes.Id, mErr *mlevelerrors.MethodLevelError) {
	panic("not implemented") // TODO: Implement
}

func (jas JAccountStub) QueryEmail(ctx context.Context, filter *basetypes.Filter, sort []basetypes.Comparator, position basetypes.Int, anchor *basetypes.Id, anchorOffset basetypes.Int, limit int, calculateTotal bool, collapseThreads bool) (queryState string, canCalculateChanges bool, retPosition basetypes.Int, ids []basetypes.Id, total basetypes.Uint, mErr *mlevelerrors.MethodLevelError) {
	panic("not implemented") // TODO: Implement
}

func (jas JAccountStub) SetEmail(ctx context.Context, ifInState *string, create map[basetypes.Id]interface{}, update map[basetypes.Id]basetypes.PatchObject, destroy []basetypes.Id) (oldState *string, newState string, created map[basetypes.Id]interface{}, updated map[basetypes.Id]interface{}, destroyed map[basetypes.Id]interface{}, notCreated map[basetypes.Id]mlevelerrors.SetError, notUpdated map[basetypes.Id]mlevelerrors.SetError, notDestroyed map[basetypes.Id]mlevelerrors.SetError, mErr *mlevelerrors.MethodLevelError) {
	panic("not implemented") // TODO: Implement
}

// Thread
func (jas JAccountStub) GetThread(ctx context.Context, ids []basetypes.Id) (state string, result []jaccount.Thread, notFound []basetypes.Id, mErr *mlevelerrors.MethodLevelError) {
	panic("not implemented") // TODO: Implement
}

func (jas JAccountStub) Close() error {
	return nil
}
