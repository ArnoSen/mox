package mailcapability

import (
	"context"

	"github.com/mjl-/mox/jmapserver/basetypes"
	"github.com/mjl-/mox/jmapserver/jaccount"
	"github.com/mjl-/mox/jmapserver/mlevelerrors"
)

type MailboxDT struct {
	//contextUserKey is the key in the context containing the user object
}

func NewMailBox() MailboxDT {
	return MailboxDT{}
}

func (m MailboxDT) Name() string {
	return "Mailbox"
}

// https://datatracker.ietf.org/doc/html/rfc8620#section-5.1
func (mb MailboxDT) Get(ctx context.Context, jaccount jaccount.JAccounter, accountId basetypes.Id, ids []basetypes.Id, properties []string, customParams any) (retAccountId basetypes.Id, state string, list []interface{}, notFound []basetypes.Id, mErr *mlevelerrors.MethodLevelError) {
	retAccountId = accountId

	mailboxes, notFound, state, mErr := jaccount.Mailbox().Get(ctx, ids)

	for _, mb := range mailboxes {
		//FIXME do not filtering on properties
		list = append(list, mb)
	}

	if notFound == nil {
		//notFound cannot be null
		notFound = []basetypes.Id{}
	}

	return accountId, state, list, notFound, mErr
}

// https://datatracker.ietf.org/doc/html/rfc8620#section-5.2
func (mb MailboxDT) Changes(ctx context.Context, jaccount jaccount.JAccounter, accountId basetypes.Id, sinceState string, maxChanges *basetypes.Uint) (retAccountId basetypes.Id, oldState string, newState string, hasMoreChanges bool, created []basetypes.Id, updated []basetypes.Id, destroyed []basetypes.Id, mErr *mlevelerrors.MethodLevelError) {
	//TODO need to add modseq for mailboxes
	//AO: not sure what to send back with regards to oldstate/newstate
	mErr = mlevelerrors.NewMethodLevelErrorCannotCalculateChanges()
	return
}

func (m MailboxDT) CustomGetRequestParams() any {
	return nil
}
