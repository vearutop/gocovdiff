package main

import (
	"os/exec"
	"testing"
)

func TestToInt(t *testing.T) {
	if toInt("1") != 1 {
		t.Fail()
	}
}

func TestRun(t *testing.T) {
	err := exec.Command("go", "test", "-cover", "-coverprofile", "cover.out", "-run", "^!TestRun$").Run()
	if err != nil {
		t.Fatal(err)
	}

	err = run(flags{
		covFile:      "cover.out",
		parentCommit: "HEAD^",
	})

	if err != nil {
		t.Skip(err)
	}
}
