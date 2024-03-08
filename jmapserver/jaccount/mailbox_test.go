package jaccount

import (
	"context"
	"testing"

	"log/slog"

	"github.com/mjl-/mox/jmapserver/basetypes"
	"github.com/mjl-/mox/mlog"
	"github.com/mjl-/mox/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMailboxes(t *testing.T) {
	t.Run("Only selected mailboxes are returned when ids is not null", func(t *testing.T) {

		//FIXME remove mock from here and just setup a bstore db on a temp path

		mbr := NewMailboxRepoMock()

		mbr.On("List").Return([]store.Mailbox{
			{
				ID: 1,
			},
			{
				ID: 2,
			},
			{
				ID: 3,
			},
		}, nil)

		ja := NewJAccount(&store.Account{
			DB: nil,
		}, mbr, mlog.New("test", slog.Default()))

		result, _, _, mErr := ja.Mailbox().Get(context.Background(), []basetypes.Id{"1", "2"})
		require.Nil(t, mErr)

		assert.Equal(t, 2, len(result))

	})
}
