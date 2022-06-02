package main

import "testing"

func TestToInt(t *testing.T) {
	if toInt("1") != 1 {
		t.Fail()
	}
}
