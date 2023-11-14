package jaccount

import (
	"reflect"
	"testing"
)

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

func AssertEqualString(t *testing.T, expected, actual string) bool {
	if expected != actual {
		t.Helper()
		t.Logf("was expecting %q but got %q", expected, actual)
		t.Fail()
	}
	return true
}

func AssertEqualInt(t *testing.T, expected, actual int) bool {
	if expected != actual {
		t.Helper()
		t.Logf("was expecting %d but got %d", expected, actual)
		t.Fail()
	}
	return true
}
