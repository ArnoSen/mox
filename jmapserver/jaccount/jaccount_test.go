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
			if parentID := NewJMailboxes(jmb1).ParentID(jmb1); parentID != nil {
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
			if parentID := NewJMailboxes(jmb1, jmb2, jmb3).ParentID(jmb3); parentID == nil {
				t.Log("was expecting a non nil parentID")
				t.FailNow()
			}
			if parentID := NewJMailboxes(jmb1, jmb2, jmb3).ParentID(jmb3); *parentID != "11" {
				t.Logf("was expecting 11 but got %s", *parentID)
				t.FailNow()
			}
		})
	})
	t.Run("ShareAncestore", func(t *testing.T) {
		t.Run("TwoTopLevel", func(t *testing.T) {
			jmb1 := NewJMailbox(store.Mailbox{
				ID:   10,
				Name: "Inbox",
			})
			jmb2 := NewJMailbox(store.Mailbox{
				ID:   11,
				Name: "Sent",
			})
			if NewJMailboxes(jmb1, jmb2).ShareAncestor(jmb1, jmb2) {
				t.Log("was expecting false but true")
				t.FailNow()
			}
		})
		t.Run("SingleMailbox", func(t *testing.T) {
			jmb1 := NewJMailbox(store.Mailbox{
				ID:   10,
				Name: "Inbox|Done",
			})
			jmb2 := NewJMailbox(store.Mailbox{
				ID:   11,
				Name: "Inbox",
			})
			if !NewJMailboxes(jmb1, jmb2).ShareAncestor(jmb1, jmb2) {
				t.Log("was expecting true but got false")
				t.FailNow()
			}
		})
	})
	t.Run("HasSpecialParent", func(t *testing.T) {
		jmb1 := NewJMailbox(store.Mailbox{
			ID:   10,
			Name: "Sent",
			SpecialUse: store.SpecialUse{
				Sent: true,
			},
		})
		jmb2 := NewJMailbox(store.Mailbox{
			ID:   11,
			Name: "Sent|Done",
		})
		jmb3 := NewJMailbox(store.Mailbox{
			ID:   12,
			Name: "Sent|Done|2023",
		})
		jmb4 := NewJMailbox(store.Mailbox{
			ID:   13,
			Name: "Custom",
		})

		jmbs := NewJMailboxes(jmb1, jmb2, jmb3, jmb4)

		if jmbs.HasSpecialParent(jmb4) {
			t.Log("was expecting false but got true")
			t.FailNow()
		}
		if !jmbs.HasSpecialParent(jmb3) {
			t.Log("was expecting true but got false")
			t.FailNow()
		}
		if !jmbs.HasSpecialParent(jmb2) {
			t.Log("was expecting true but got false")
			t.FailNow()
		}

	})
	/*
		This needs to be fixed
			t.Run("Sort", func(t *testing.T) {
				t.Run("SpecialMailboxesDoNotChangeOrder", func(t *testing.T) {
					t.Run("Test1", func(t *testing.T) {
						jmb1 := NewJMailbox(store.Mailbox{
							ID:   10,
							Name: "Archive",
							SpecialUse: store.SpecialUse{
								Archive: true,
							},
						})
						jmb2 := NewJMailbox(store.Mailbox{
							ID:   11,
							Name: "Sent",
							SpecialUse: store.SpecialUse{
								Sent: true,
							},
						})
						jmb3 := NewJMailbox(store.Mailbox{
							ID:   12,
							Name: "Junk",
							SpecialUse: store.SpecialUse{
								Junk: true,
							},
						})
						jmb4 := NewJMailbox(store.Mailbox{
							ID:         13,
							Name:       "Custom",
							SpecialUse: store.SpecialUse{},
						})
						jmbs := NewJMailboxes(jmb1, jmb2, jmb3, jmb4)
						sort.Stable(jmbs)
						if id := jmbs.Mbs[0].ID(); id != "10" {
							t.Logf("was expecting 10 but got %s", id)
							t.FailNow()
						}
						if id := jmbs.Mbs[1].ID(); id != "11" {
							t.Logf("was expecting 11 but got %s", id)
							t.FailNow()
						}
						if id := jmbs.Mbs[2].ID(); id != "12" {
							t.Logf("was expecting 12 but got %s", id)
							t.FailNow()
						}
						if id := jmbs.Mbs[3].ID(); id != "13" {
							t.Logf("was expecting 13 but got %s", id)
							t.FailNow()
						}
					})
					t.Run("Test2", func(t *testing.T) {
						jmb1 := NewJMailbox(store.Mailbox{
							ID:   10,
							Name: "Sent",
							SpecialUse: store.SpecialUse{
								Sent: true,
							},
						})
						jmb2 := NewJMailbox(store.Mailbox{
							ID:   11,
							Name: "Archive",
							SpecialUse: store.SpecialUse{
								Archive: true,
							},
						})
						jmb3 := NewJMailbox(store.Mailbox{
							ID:   12,
							Name: "Junk",
							SpecialUse: store.SpecialUse{
								Junk: true,
							},
						})
						jmb4 := NewJMailbox(store.Mailbox{
							ID:         13,
							Name:       "Custom",
							SpecialUse: store.SpecialUse{},
						})
						jmbs := NewJMailboxes(jmb1, jmb2, jmb3, jmb4)
						sort.Stable(jmbs)
						if id := jmbs.Mbs[0].ID(); id != "10" {
							t.Logf("was expecting 10 but got %s", id)
							t.FailNow()
						}
						if id := jmbs.Mbs[1].ID(); id != "11" {
							t.Logf("was expecting 11 but got %s", id)
							t.FailNow()
						}
						if id := jmbs.Mbs[2].ID(); id != "12" {
							t.Logf("was expecting 12 but got %s", id)
							t.FailNow()
						}
						if id := jmbs.Mbs[3].ID(); id != "13" {
							t.Logf("was expecting 13 but got %s", id)
							t.FailNow()
						}
					})
					t.Run("SpecialMailboxesAreReturnedInTheOrderTheCome", func(t *testing.T) {
						jmb1 := NewJMailbox(store.Mailbox{
							ID:   10,
							Name: "Sent",
							SpecialUse: store.SpecialUse{
								Sent: true,
							},
						})
						jmb2 := NewJMailbox(store.Mailbox{
							ID:   11,
							Name: "Archive",
							SpecialUse: store.SpecialUse{
								Archive: true,
							},
						})
						jmb3 := NewJMailbox(store.Mailbox{
							ID:   12,
							Name: "Junk",
							SpecialUse: store.SpecialUse{
								Junk: true,
							},
						})
						jmb4 := NewJMailbox(store.Mailbox{
							ID:         13,
							Name:       "Custom",
							SpecialUse: store.SpecialUse{},
						})
						jmbs := NewJMailboxes(jmb1, jmb2, jmb3, jmb4)
						sort.Stable(jmbs)
						if id := jmbs.Mbs[0].ID(); id != "10" {
							t.Logf("was expecting 10 but got %s", id)
							t.FailNow()
						}
						if id := jmbs.Mbs[1].ID(); id != "11" {
							t.Logf("was expecting 11 but got %s", id)
							t.FailNow()
						}
						if id := jmbs.Mbs[2].ID(); id != "12" {
							t.Logf("was expecting 12 but got %s", id)
							t.FailNow()
						}
						if id := jmbs.Mbs[3].ID(); id != "13" {
							t.Logf("was expecting 13 but got %s", id)
							t.FailNow()
						}
					})
					t.Run("CustomMailboxesGoAfterSpecialMailboxes", func(t *testing.T) {
						jmb1 := NewJMailbox(store.Mailbox{
							ID:         10,
							Name:       "Custom",
							SpecialUse: store.SpecialUse{},
						})
						jmb2 := NewJMailbox(store.Mailbox{
							ID:   11,
							Name: "Sent",
							SpecialUse: store.SpecialUse{
								Sent: true,
							},
						})
						jmb3 := NewJMailbox(store.Mailbox{
							ID:   12,
							Name: "Archive",
							SpecialUse: store.SpecialUse{
								Archive: true,
							},
						})
						jmb4 := NewJMailbox(store.Mailbox{
							ID:   13,
							Name: "Junk",
							SpecialUse: store.SpecialUse{
								Junk: true,
							},
						})
						jmbs := NewJMailboxes(jmb1, jmb2, jmb3, jmb4)
						sort.Stable(jmbs)
						if id := jmbs.Mbs[0].ID(); id != "11" {
							t.Logf("was expecting 11 but got %s", id)
							t.FailNow()
						}
						if id := jmbs.Mbs[1].ID(); id != "12" {
							t.Logf("was expecting 12 but got %s", id)
							t.FailNow()
						}
						if id := jmbs.Mbs[2].ID(); id != "13" {
							t.Logf("was expecting 13 but got %s", id)
							t.FailNow()
						}
						if id := jmbs.Mbs[3].ID(); id != "10" {
							t.Logf("was expecting 10 but got %s", id)
							t.FailNow()
						}
					})
				})

				t.Run("MailboxesAreSortedByDepthFirstByNameSecond", func(t *testing.T) {
					jmb1 := NewJMailbox(store.Mailbox{
						ID:         10,
						Name:       "NewTopLevel",
						SpecialUse: store.SpecialUse{},
					})
					jmb2 := NewJMailbox(store.Mailbox{
						ID:   11,
						Name: "Inbox",
						SpecialUse: store.SpecialUse{
							Archive: true,
						},
					})
					jmb3 := NewJMailbox(store.Mailbox{
						ID:         12,
						Name:       "Inbox|2023|Done",
						SpecialUse: store.SpecialUse{},
					})
					jmb4 := NewJMailbox(store.Mailbox{
						ID:         13,
						Name:       "Inbox|2023",
						SpecialUse: store.SpecialUse{},
					})
					jmbs := NewJMailboxes(jmb1, jmb2, jmb3, jmb4)
					sort.Stable(jmbs)
					if id := jmbs.Mbs[0].ID(); id != "11" {
						t.Logf("was expecting 11 but got %s", id)
						t.FailNow()
					}
					if id := jmbs.Mbs[1].ID(); id != "13" {
						t.Logf("was expecting 13 but got %s", id)
						t.FailNow()
					}
					if id := jmbs.Mbs[2].ID(); id != "12" {
						t.Logf("was expecting 12 but got %s", id)
						t.FailNow()
					}
					if id := jmbs.Mbs[3].ID(); id != "10" {
						t.Logf("was expecting 10 but got %s", id)
						t.FailNow()
					}
				})

				t.Run("SubMailboxesOfSpecialMailboxesAreKeptTogehter", func(t *testing.T) {
					jmb1 := NewJMailbox(store.Mailbox{
						ID:   10,
						Name: "Sent",
						SpecialUse: store.SpecialUse{
							Sent: true,
						},
					})
					jmb2 := NewJMailbox(store.Mailbox{
						ID:   11,
						Name: "Junk",
						SpecialUse: store.SpecialUse{
							Junk: true,
						},
					})
					jmb3 := NewJMailbox(store.Mailbox{
						ID:   12,
						Name: "Sent|2023",
					})
					jmb4 := NewJMailbox(store.Mailbox{
						ID:   13,
						Name: "Junk|2023",
					})
					jmbs := NewJMailboxes(jmb1, jmb2, jmb3, jmb4)
					sort.Stable(jmbs)
					if id := jmbs.Mbs[0].ID(); id != "10" {
						t.Logf("was expecting 10 but got %s", id)
						t.FailNow()
					}
					if id := jmbs.Mbs[1].ID(); id != "12" {
						t.Logf("was expecting 12 but got %s", id)
						t.FailNow()
					}
					if id := jmbs.Mbs[2].ID(); id != "11" {
						t.Logf("was expecting 11 but got %s", id)
						t.FailNow()
					}
					if id := jmbs.Mbs[3].ID(); id != "13" {
						t.Logf("was expecting 13 but got %s", id)
						t.FailNow()
					}
				})
			})
	*/
}
