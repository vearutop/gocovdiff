# gocovdiff

[![Build Status](https://github.com/vearutop/gocovdiff/workflows/test-unit/badge.svg)](https://github.com/vearutop/gocovdiff/actions?query=branch%3Amaster+workflow%3Atest-unit)
[![Coverage Status](https://codecov.io/gh/vearutop/gocovdiff/branch/master/graph/badge.svg)](https://codecov.io/gh/vearutop/gocovdiff)

A tool to annotate Go code coverage for changed statements in GitHub pull requests.

## Why?

When code is changed or introduced in a pull request, it is often difficult to find out if changed statements are 
sufficiently covered with tests. 

> Make sure that frequently changing code is covered. While project wide goals above 90% are most likely not worth it, per-commit coverage goals of 99% are reasonable, and 90% is a good lower threshold. We need to ensure that our tests are not getting worse over time.

https://testing.googleblog.com/2020/08/code-coverage-best-practices.html

This tool analyzes changed lines (derived from `git diff`) against test coverage data and counts coverage ratio only in changed lines.
The result is present as annotations pointing to uncovered lines and a summary grouped by file and function.

There is a caveat, such approach would not show coverage change if a test was added or updated, but the tested code was not changed.
This case can be handled by reporting global function coverage diff against base (`-func-base-cov` and `-func-cov`). 

## Install

```
go install github.com/vearutop/gocovdiff@latest
$(go env GOPATH)/bin/gocovdiff --help
```

Or download binary from [releases](https://github.com/vearutop/gocovdiff/releases).

## Usage

```
gocovdiff -help
Usage of gocovdiff:
  -cov string
        Coverage file (default "coverage.txt")
  -delta-cov-file string
        File to store delta coverage message
  -diff string
        Git diff file for changes (optional)
  -exclude string
        Exclude directories by prefix and files by name pattern, comma separated (optional)
  -func-base-cov string
        Base func coverage from 'go tool cover -func', requires -func-cov (optional)
  -func-cov string
        Current func coverage from 'go tool cover -func', requires -func-base-cov or -func-max-cov (optional)
  -func-max-cov float
        Max func coverage from 'go tool cover -func' to keep in report of undercovered functions, requires -func-cov (optional)
  -gha-annotations string
        File to store GitHub Actions annotations
  -mod string
        Module name (optional)
  -parent string
        Parent commit hash (optional)
  -target-delta-cov float
        Target coverage of changed lines, to be used together with -delta-cov-file (default 80)
  -version
        Show version and exit
```

## GitHub Action

This tool can produce GitHub Actions annotations to mark changed lines of code that were not covered with tests.

![Annotations](./resources/annotations.png)

Also, you can comment on the pull request with the report.

![Comment](./resources/comment.png)

```
      - name: Test
        id: test
        run: |
          make test-unit

      - name: Annotate missing test coverage
        id: annotate
        if: github.event.pull_request.base.sha != ''
        run: |
          git fetch origin master ${{ github.event.pull_request.base.sha }}
          curl -sLO https://github.com/vearutop/gocovdiff/releases/download/v1.3.4/linux_amd64.tar.gz && tar xf linux_amd64.tar.gz && echo "b351c67526eefeb0671c82e9271ae984875865eed19e911f40f78348cb98347c  gocovdiff" | shasum -c
          REP=$(./gocovdiff -cov unit.coverprofile -gha-annotations gha-unit.txt)
          echo "${REP}"
          REP="${REP//$'\n'/%0A}"
          cat gha-unit.txt
          echo "::set-output name=rep::$REP"
      - name: Comment Test Coverage
        continue-on-error: true
        if: github.event.pull_request.base.sha != ''
        uses: marocchino/sticky-pull-request-comment@v2
        with:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          header: unit-test
          message: |
            ### Unit Test Coverage
            <details><summary>Coverage of changed lines</summary>
            
            ${{ steps.annotate.outputs.rep }}
            </details>

```

[Workflow example](https://github.com/bool64/dev/blob/v0.2.15/templates/github/workflows/test-unit.yml).


## Example 
```
make test && gocovdiff -cov unit.coverprofile
```
```
Running unit tests.
ok      github.com/vearutop/gocovdiff   0.738s  coverage: 71.9% of statements
|           File           |      Function       | Coverage |
|--------------------------|---------------------|----------|
| Total                    |                     | 71.86%   |
| diff.go                  |                     | 62.50%   |
| diff.go:13               | gitDiff             | 60.00%   |
| diff.go:37               | getDiff             | 57.14%   |
| func.go                  |                     | 92.31%   |
| func.go:52               | Visit               | 100.00%  |
| func.go:16               | findFuncs           | 85.71%   |
| git.go                   |                     | 35.29%   |
| git.go:11                | forkPointFromLocal  | 75.00%   |
| git.go:27                | forkPointFromGitHub | 0.00%    |
| github_annotations.go    |                     | 20.00%   |
| github_annotations.go:24 | printNotice         | 40.00%   |
| github_annotations.go:13 | printNotTested      | 0.00%    |
| gocovdiff.go             |                     | 74.11%   |
| gocovdiff.go:49          | run                 | 82.00%   |
| gocovdiff.go:20          | parseFlags          | 0.00%    |
| gocovdiff.go:43          | main                | 0.00%    |
| profile.go               |                     | 80.00%   |
| profile.go:25            | parseProfiles       | 76.92%   |
| profile.go:86            | toInt               | 75.00%   |
| report.go                |                     | 96.00%   |
| report.go:11             | printReport         | 92.00%   |
```

### Format func coverage diff against base coverage

```
git checkout master && make test && go tool cover -func=unit.coverprofile > base.func.txt 
git checkout my-branch && make test && go tool cover -func=unit.coverprofile > cur.func.txt
gocovdiff -func-cov cur.func.txt -func-base-cov base.func.txt
```

```
|     File      | Function | Base Coverage | Current Coverage |
|---------------|----------|---------------|------------------|
| Total         |          | 70.0%         | 56.2% (-13.80%)  |
| sample/bar.go | Bar      | 80.0%         | 71.4% (-8.60%)   |
| sample/foo.go | foo      | 60.0%         | 44.4% (-15.60%)  |
```

### Filter under covered functions from func coverage

```
go tool cover -func=unit.coverprofile > cur.func.txt
gocovdiff -func-cov cur.func.txt -func-max-cov 70
```

```
|     File      | Function | Coverage |
|---------------|----------|----------|
| sample/foo.go | foo      | 44.4%    |
```
