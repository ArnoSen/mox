package jaccount

import "github.com/mjl-/mox/store"

type MailboxRepo interface {
	List() ([]store.Mailbox, error)
}
