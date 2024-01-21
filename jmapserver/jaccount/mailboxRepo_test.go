package jaccount

import (
	"github.com/mjl-/mox/store"
	"github.com/stretchr/testify/mock"
)

type MailboxRepoMock struct {
	mock.Mock
}

func NewMailboxRepoMock() MailboxRepoMock {
	return MailboxRepoMock{}
}

func (mrm MailboxRepoMock) List() ([]store.Mailbox, error) {
	args := mrm.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]store.Mailbox), args.Error(1)
}
