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
		deltaCovFile:   "delta.txt",
		targetDeltaCov: 81.5,
	}, report))

	assert.Equal(t, `|   File   | Function | Coverage |
|----------|----------|----------|
| Total    |          | 33.33%   |
| bar.go   |          | 50.00%   |
| bar.go:3 | Bar      | 0.00%    |
| foo.go   |          | 25.00%   |
| foo.go:5 | foo      | 0.00%    |
`, report.String())

	gha, err := ioutil.ReadFile("gha.txt")
	require.NoError(t, err)

	assert.Equal(t, `bar.go:8,10: 2 statement(s) not covered by tests
::notice file=bar.go,line=8,endLine=10::2 statement(s) not covered by tests.
foo.go:6,8: 2 statement(s) not covered by tests
::notice file=foo.go,line=6,endLine=8::2 statement(s) not covered by tests.
foo.go:18,20: 2 statement(s) not covered by tests
::notice file=foo.go,line=18,endLine=20::2 statement(s) not covered by tests.
`, string(gha))

	delta, err := ioutil.ReadFile("delta.txt")
	require.NoError(t, err)

	assert.Equal(t, "changed lines: (statements) 33.33% (coverage is less than 81.50%, consider testing the changes more thoroughly)", string(delta))
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

	println(report.String())

	assert.Equal(t, `|     File      | Function | Base Coverage | Current Coverage |
|---------------|----------|---------------|------------------|
| Total         |          | 70.0%         | 56.2% (-13.80%)  |
| sample/bar.go | Bar      | 80.0%         | 71.4% (-8.60%)   |
| sample/foo.go | foo      | 60.0%         | 44.4% (-15.60%)  |
`, report.String())
}

func TestRun_funcUndercovered(t *testing.T) {
	require.NoError(t, os.Chdir("_testdata"))

	defer func() {
		require.NoError(t, os.Chdir(".."))
	}()

	report := bytes.NewBuffer(nil)

	require.NoError(t, run(flags{
		funcCov:    "cur.func.txt",
		funcMaxCov: 70,
	}, report))

	assert.Equal(t, `|     File      | Function | Coverage |
|---------------|----------|----------|
| sample/foo.go | foo      | 44.4%    |
`, report.String())
}
