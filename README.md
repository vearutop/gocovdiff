# gocovdiff

[![Build Status](https://github.com/vearutop/gocovdiff/workflows/test-unit/badge.svg)](https://github.com/vearutop/gocovdiff/actions?query=branch%3Amaster+workflow%3Atest-unit)
[![Coverage Status](https://codecov.io/gh/vearutop/gocovdiff/branch/master/graph/badge.svg)](https://codecov.io/gh/vearutop/gocovdiff)

A tool to annotate Go code coverage in GitHub pull requests.

## Usage

```
      - name: Annotate missing test coverage
        if: ${{ github.event.pull_request.base.sha }}
        run: |
          go install .
          git fetch origin master ${{ github.event.pull_request.base.sha }}
          git diff ${{ github.event.pull_request.base.sha }} > diff.txt
          gocovdiff -diff diff.txt -cov unit.coverprofile -mod $(go list -m)

```