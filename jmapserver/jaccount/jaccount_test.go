package jaccount

import (
	"testing"

	"github.com/mjl-/mox/store"
)

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
