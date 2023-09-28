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
func (mb MailboxDT) Get(ctx context.Context, jaccount jaccount.JAccounter, accountId basetypes.Id, ids []basetypes.Id, properties []string) (retAccountId basetypes.Id, state string, list []interface{}, notFound []basetypes.Id, mErr *mlevelerrors.MethodLevelError) {
	retAccountId = accountId

	mailboxes, notFound, state, mErr := jaccount.GetMailboxes(ctx, ids)

	for _, mb := range mailboxes {
		//FIXME do not filtering on properties
		list = append(list, mb)
	}

	return accountId, state, list, notFound, mErr
}
