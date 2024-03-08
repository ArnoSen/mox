package mailcapability

import (
	"context"
	"log/slog"
	"testing"

	"github.com/mjl-/mox/jmapserver/basetypes"
	"github.com/mjl-/mox/jmapserver/jaccount"
	"github.com/mjl-/mox/jmapserver/testutils"
	"github.com/mjl-/mox/mlog"
	"github.com/mjl-/mox/store"
	"github.com/stretchr/testify/require"
)

func TestGetMailboxes(t *testing.T) {
	t.Run("Get", func(t *testing.T) {
		t.Run("Only selected mailboxes are returned when ids is not null", func(t *testing.T) {

			testDB, err := testutils.GetTestDB(store.Mailbox{})
			testutils.RequireNoError(t, err)
			defer testDB.Close()
			testutils.RequireNoError(t, testDB.DB.Insert(context.Background(), &store.Mailbox{
				Name: "m1",
				ID:   1,
			}))
			testutils.RequireNoError(t, testDB.DB.Insert(context.Background(), &store.Mailbox{
				Name: "m2",
				ID:   2,
			}))
			testutils.RequireNoError(t, testDB.DB.Insert(context.Background(), &store.Mailbox{
				Name: "m3",
				ID:   3,
			}))

			ja := jaccount.NewJAccount(&store.Account{
				DB: testDB.DB,
			}, mlog.New("test", slog.Default()))

			mailboxDT := NewMailBox(mlog.New("test", slog.Default()))

			_, _, result, _, mErr := mailboxDT.Get(context.Background(), ja, basetypes.Id("id"), []basetypes.Id{"1", "2"}, nil, nil)
			require.Nil(t, mErr)

			testutils.AssertEqual(t, 2, len(result))
		})
	})
}

// FIXME need to get a test for getting the parent and the sort id
func TestJMailBoxes(t *testing.T) {
	t.Run("ParentID", func(t *testing.T) {
		t.Run("SingleMailbox", func(t *testing.T) {
			jmb1 := NewJMailbox(store.Mailbox{
				ID:   10,
				Name: "Inbox",
				SpecialUse: store.SpecialUse{
					Archive: true,
				},
			})
			if parentID := NewJMailboxes("|", jmb1).ParentID(jmb1); parentID != nil {
				t.Logf("was expecting nil but got %s", *parentID)
				t.FailNow()
			}
		})
		t.Run("Multiple", func(t *testing.T) {
			jmb1 := NewJMailbox(store.Mailbox{
				ID:   10,
				Name: "Inbox",
			})
			jmb2 := NewJMailbox(store.Mailbox{
				ID:   11,
				Name: "Inbox|2023",
			})
			jmb3 := NewJMailbox(store.Mailbox{
				ID:   12,
				Name: "Inbox|2023|Done",
			})
			if parentID := NewJMailboxes("|", jmb1, jmb2, jmb3).ParentID(jmb3); parentID == nil {
				t.Log("was expecting a non nil parentID")
				t.FailNow()
			}
			if parentID := NewJMailboxes("|", jmb1, jmb2, jmb3).ParentID(jmb3); *parentID != "11" {
				t.Logf("was expecting 11 but got %s", *parentID)
				t.FailNow()
			}
		})
	})
}
