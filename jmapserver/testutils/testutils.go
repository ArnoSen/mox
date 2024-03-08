package testutils

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/mjl-/bstore"
)

type TestDB struct {
	DB  *bstore.DB
	dir string
}

func (t TestDB) Close() error {
	t.DB.Close()
	return os.RemoveAll(t.dir) // clean upkkk

}

func GetTestDB(datatypes ...any) (*TestDB, error) {
	dir, err := os.MkdirTemp("", "testmailchanges")
	if err != nil {
		return nil, err
	}
	db, err := bstore.Open(context.Background(), filepath.Join(dir, "mydb.db"), nil, datatypes...)
	if err != nil {
		defer os.RemoveAll(dir) // clean upkkk
		return nil, err
	}
	return &TestDB{
		dir: dir,
		DB:  db,
	}, nil

}

func RequireNoError(t *testing.T, e error) {
	if !(e == nil || reflect.ValueOf(e).IsNil()) {
		t.Helper()
		t.Logf("was expecting no error but got %s", e.Error())
		t.FailNow()
	}
}

func AssertNil(t *testing.T, i any) bool {
	if i == nil || reflect.ValueOf(i).IsNil() {
		return true
	}

	t.Helper()
	t.Logf("was expecting nil but got %+v", i)
	t.Fail()
	return false
}

func AssertNotNil(t *testing.T, i any) bool {
	if i == nil {
		t.Logf("was expecting not nil but nil")
		t.Fail()
		return false
	}
	return true
}

func AssertTrue(t *testing.T, b bool) bool {
	if !b {
		t.Helper()
		t.Logf("was expecting true but got false")
		t.Fail()
	}
	return b
}

func AssertEqual[V comparable](t *testing.T, expected, actual V) bool {
	if expected != actual {
		t.Helper()
		t.Logf("was expecting %v but got %v", expected, actual)
		t.Fail()
	}
	return true
}
