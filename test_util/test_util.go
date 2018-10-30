package test_util

import (
	"reflect"
	"testing"
)

func Assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	tb.Helper()
	if !condition {
		tb.Logf(msg, v...)
		tb.FailNow()
	}
}

func Ok(tb testing.TB, err error) {
	tb.Helper()
	if err != nil {
		tb.Error(err)
		tb.FailNow()
	}
}

func Equals(tb testing.TB, expected interface{}, actual interface{}) {
	tb.Helper()
	if !reflect.DeepEqual(expected, actual) {
		msg := "expected %#v:\n\nactual: %#v"
		tb.Logf(msg, expected, actual)
		tb.FailNow()
	}
}
