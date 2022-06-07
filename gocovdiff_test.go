package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"
)

func TestToInt(t *testing.T) {
	if toInt("1") != 1 {
		t.Fail()
	}
}

func TestRun(t *testing.T) {
	err := os.Chdir("_testdata")
	if err != nil {
		t.Fatal(err)
	}

	report := bytes.NewBuffer(nil)

	err = run(flags{
		diffFile:       "diff.txt",
		covFile:        "coverage.txt",
		ghaAnnotations: "gha.txt",
	}, report)

	if err != nil {
		t.Fatal(err)
	}

	if report.String() != `|   File   | Function | Coverage |
|----------|----------|----------|
| Total    |          | 53.85%   |
| bar.go   |          | 80.00%   |
| bar.go:3 | Bar      | 60.00%   |
| foo.go   |          | 37.50%   |
| foo.go:5 | foo      | 25.00%   |
` {
		t.Fatal("Unexpected report:\n", report.String())
	}

	gha, err := ioutil.ReadFile("gha.txt")
	if err != nil {
		t.Fatal(err)
	}

	if string(gha) != `::notice file=bar.go,line=8,endLine=10::2 statement(s) not covered by tests.
::notice file=foo.go,line=6,endLine=8::1 statement(s) not covered by tests.
::notice file=foo.go,line=10,endLine=12::2 statement(s) not covered by tests.
::notice file=foo.go,line=18,endLine=20::2 statement(s) not covered by tests.
::notice file=foo.go,line=22,endLine=22::1 statement(s) not covered by tests.
` {
		t.Fatal("Unexpected annotations:\n", string(gha))
	}
}
