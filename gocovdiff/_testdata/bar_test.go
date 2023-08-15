package sample_test

import (
	"testing"
)

func TestBar(t *testing.T) {
	if Bar(1) {
		t.Fail()
	}

	if !Bar(11) {
		t.Fail()
	}
}
