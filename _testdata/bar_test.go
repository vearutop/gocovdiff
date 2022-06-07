package sample_test

import (
	"sample"
	"testing"
)

func TestBar(t *testing.T) {
	if sample.Bar(1) {
		t.Fail()
	}

	if !sample.Bar(11) {
		t.Fail()
	}
}
