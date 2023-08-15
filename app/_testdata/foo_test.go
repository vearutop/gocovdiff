package sample

import "testing"

func TestFoo(t *testing.T) {
	if !foo(11) {
		t.Fail()
	}
}
