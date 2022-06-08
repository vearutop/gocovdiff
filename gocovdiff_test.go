package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToInt(t *testing.T) {
	if toInt("1") != 1 {
		t.Fail()
	}
}

func TestRun(t *testing.T) {
	require.NoError(t, os.Chdir("_testdata"))

	defer func() {
		require.NoError(t, os.Chdir(".."))
	}()

	report := bytes.NewBuffer(nil)

	require.NoError(t, run(flags{
		diffFile:       "diff.txt",
		covFile:        "coverage.txt",
		ghaAnnotations: "gha.txt",
	}, report))

	assert.Equal(t, `|   File   | Function | Coverage |
|----------|----------|----------|
| Total    |          | 53.85%   |
| bar.go   |          | 80.00%   |
| bar.go:3 | Bar      | 60.00%   |
| foo.go   |          | 37.50%   |
| foo.go:5 | foo      | 25.00%   |
`, report.String())

	gha, err := ioutil.ReadFile("gha.txt")
	require.NoError(t, err)

	assert.Equal(t, `bar.go:8,10: 2 statement(s) not covered by tests
::notice file=bar.go,line=8,endLine=10::2 statement(s) not covered by tests.
foo.go:6,8: 1 statement(s) not covered by tests
::notice file=foo.go,line=6,endLine=8::1 statement(s) not covered by tests.
foo.go:10,12: 2 statement(s) not covered by tests
::notice file=foo.go,line=10,endLine=12::2 statement(s) not covered by tests.
foo.go:18,20: 2 statement(s) not covered by tests
::notice file=foo.go,line=18,endLine=20::2 statement(s) not covered by tests.
foo.go:22,22: 1 statement(s) not covered by tests
::notice file=foo.go,line=22,endLine=22::1 statement(s) not covered by tests.
`, string(gha))
}

func TestRun_funcCov(t *testing.T) {
	require.NoError(t, os.Chdir("_testdata"))

	defer func() {
		require.NoError(t, os.Chdir(".."))
	}()

	report := bytes.NewBuffer(nil)

	require.NoError(t, run(flags{
		funcCov:     "cur.func.txt",
		funcBaseCov: "base.func.txt",
	}, report))

	assert.Equal(t, `|     File      | Function | Base Coverage | Current Coverage |
|---------------|----------|---------------|------------------|
| Total         |          | 70.0          | 56.2             |
| sample/bar.go | Bar      | 80.0          | 71.4             |
| sample/foo.go | foo      | 60.0          | 44.4             |
`, report.String())
}
